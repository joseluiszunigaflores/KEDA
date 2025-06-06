# Escalado Automático de Consumidores Kafka con KEDA en Kubernetes (Go)

Este proyecto demuestra cómo desplegar un clúster de Kafka en Kubernetes y escalar automáticamente un consumidor de Go basado en el lag de mensajes de Kafka, utilizando KEDA (Kubernetes Event-driven Autoscaling). Incluye un productor de Go para simular la carga de mensajes.

## Contenido del Repositorio

* `zookeeper.yaml`: Despliegue y Servicio de Zookeeper.
* `kafka.yaml`: Despliegue y Servicio de Apache Kafka.
* `golang-kafka-consumer-deployment.yaml`: Despliegue del consumidor de Go.
* `kafka-consumer-scaledobject.yaml`: Configuración de KEDA para el escalado del consumidor de Go.
* `golang-kafka-producer-job.yaml`: Job de Kubernetes para el productor de Go (simula el envío de mensajes).
* `go-producer/main.go`: Código fuente del productor de Go.
* `go-consumer/main.go`: Código fuente del consumidor de Go.

## Prerrequisitos

Antes de comenzar, asegúrate de tener lo siguiente instalado y configurado:

1.  **Kubernetes Cluster:** Un clúster de Kubernetes en funcionamiento (por ejemplo, Minikube, kind, GKE, AKS, EKS).
    * Si usas Minikube, asegúrate de que esté iniciado: `minikube start`
2.  **kubectl:** La herramienta de línea de comandos de Kubernetes.
3.  **Docker:** Para construir las imágenes de tus aplicaciones Go.
    * Si usas Minikube, configura tu entorno Docker para construir directamente en el daemon de Minikube: `eval $(minikube docker-env)`
4.  **KEDA:** El operador de KEDA debe estar instalado en tu clúster.
    * Si no lo tienes, puedes instalarlo con Helm:
        ```bash
        helm repo add keda [https://kedacore.github.io/charts](https://kedacore.github.io/charts)
        helm repo update
        kubectl create namespace keda
        helm install keda keda/keda --namespace keda
        ```

## 1. Preparar Aplicaciones Go e Imágenes Docker

Asegúrate de que tus aplicaciones Go estén configuradas para leer las variables de entorno para la conexión a Kafka.

**go-producer/main.go (Ejemplo):**
```go
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"[github.com/IBM/sarama](https://github.com/IBM/sarama)"
)

func main() {
	brokers := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")
	numMessagesStr := os.Getenv("NUM_MESSAGES")

	if brokers == "" || topic == "" || numMessagesStr == "" {
		log.Fatalf("KAFKA_BROKERS, KAFKA_TOPIC, and NUM_MESSAGES environment variables must be set.")
	}

	numMessages, err := strconv.Atoi(numMessagesStr)
	if err != nil {
		log.Fatalf("NUM_MESSAGES must be an integer: %v", err)
	}

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true
	config.ClientID = "golang-producer-client"

	producer, err := sarama.NewSyncProducer([]string{brokers}, config)
	if err != nil {
		log.Fatalf("Failed to start Sarama producer: %v", err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalf("Failed to close producer: %v", err)
		}
	}()

	for i := 0; i < numMessages; i++ {
		msgValue := fmt.Sprintf("Hello Kafka from Go! Message %d", i)
		msg := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.StringEncoder(msgValue),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("Failed to send message %d: %v", i, err)
		} else {
			fmt.Printf("Message %d sent to topic %s, partition %d, offset %d\n", i, topic, partition, offset)
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("Producer finished sending messages.")
}
```

### Construir imágenes Docker:
Navega a los directorios de tu productor y consumidor (go-producer, go-consumer) y construye las imágenes:

```bash
# Para el productor
cd go-producer
docker build -t golang-kafka-producer:1.0.0 .
cd ..

# Para el consumidor
cd go-consumer
docker build -t golang-kafka-consumer:1.0.0 .
cd ..

```


### Despliegue en Kubernetes
Sigue estos pasos en orden para desplegar todos los componentes.

#### Crear Namespace para Kafka
``` bash

kubectl create namespace kafka-ns
2.2 Desplegar Zookeeper


kubectl apply -f zookeeper.yaml -n kafka-ns



kubectl get pods -n kafka-ns -l app=zookeeper -w # Espera '1/1 Running'
kubectl logs $(kubectl get pods -n kafka-ns -l app=zookeeper -o jsonpath='{.items[0].metadata.name}') -n kafka-ns

```
#### Desplegar Kafka

``` bash
kubectl apply -f kafka.yaml -n kafka-ns
Verificación:


kubectl get pods -n kafka-ns -l app=kafka -w # Espera '1/1 Running'
kubectl logs $(kubectl get pods -n kafka-ns -l app=kafka -o jsonpath='{.items[0].metadata.name}') -n kafka-ns # Busca 'Kafka Server started'

```
#### Crear el Tópico de Kafka
Es fundamental que el tópico my-golang-topic exista antes de que los productores intenten escribir en él.

``` bash

KAFKA_POD_NAME=$(kubectl get pods -n kafka-ns -l app=kafka -o jsonpath='{.items[0].metadata.name}')
echo "Pod de Kafka: $KAFKA_POD_NAME"
## Crear el tópico:

kubectl exec -it $KAFKA_POD_NAME -n kafka-ns -- kafka-topics --create --topic my-golang-topic --bootstrap-server localhost:9092 --partitions 1 --replication-factor 1
Verás Created topic my-golang-topic.
Verificar que el tópico fue creado:


kubectl exec -it $KAFKA_POD_NAME -n kafka-ns -- kafka-topics --list --bootstrap-server localhost:9092
Deberías ver my-golang-topic en la lista.
2.5 Desplegar el Consumidor de Go


kubectl apply -f golang-kafka-consumer-deployment.yaml -n default # O tu namespace de aplicación


kubectl get deployment golang-kafka-consumer -n default # Debería mostrar '0/0' réplicas inicialmente.
```

#### Desplegar el ScaledObject de KEDA

``` bash

kubectl apply -f kafka-consumer-scaledobject.yaml -n default # Mismo namespace que el consumidor
Verificación:

kubectl get scaledobject kafka-consumer-scaledobject -n default -o yaml

```
## Observabilidad

Revisa el status: Deberías ver Ready: True y Active: True (o False si no hay actividad aún). Asegúrate de que no haya errores de campos desconocidos.


#### Observar el HPA (Horizontal Pod Autoscaler) de KEDA
Esta es la forma más directa de ver cómo KEDA detecta el lag y pide réplicas.

``` bash
#Terminal 1:
kubectl get hpa -n default -w
```

* TARGETS: Muestra (lag_actual)/(lag_threshold). Cuando el lag aumenta, este número subirá.
* REPLICAS: El número actual de pods de consumidor que el HPA está gestionando.

#### Observar el Deployment del Consumidor
Verás cómo el número de réplicas en tu Deployment de consumidor cambia.

``` bash
Terminal 2:

kubectl get deployment golang-kafka-consumer -n default -w
```
* READY: Verás cómo el número de réplicas listas (1/1, 2/2, etc.) aumenta o disminuye.
#### Observar los Pods del Consumidor

Verás los pods individuales crearse y destruirse.
``` bash
Terminal 3:

kubectl get pods -n default -l app=golang-kafka-consumer -w
```
* Verás nuevas líneas con el estado ContainerCreating -> Running cuando se escalan.
* Verás pods pasar a Terminating cuando se reducen.


# Ejemplo

``` bash

```

``` bash
kubectl get deployment golang-kafka-consumer -n default -w
NAME                    READY   UP-TO-DATE   AVAILABLE   AGE
golang-kafka-consumer   0/0     0            0           143m
golang-kafka-consumer   0/1     0            0           160m
golang-kafka-consumer   0/1     0            0           160m
golang-kafka-consumer   0/1     0            0           160m
golang-kafka-consumer   0/1     1            0           160m
golang-kafka-consumer   1/1     1            1           160m
golang-kafka-consumer   1/2     1            1           160m
golang-kafka-consumer   1/2     1            1           160m
golang-kafka-consumer   1/2     1            1           160m
golang-kafka-consumer   1/2     2            1           160m
golang-kafka-consumer   2/2     2            2           161m
golang-kafka-consumer   2/0     2            2           165m
golang-kafka-consumer   2/0     2            2           165m
golang-kafka-consumer   0/0     0            0           165m
```

``` bash
kubectl get pods -n default -w
NAME                       READY   STATUS      RESTARTS   AGE
kafka-producer-job-8c2tj   0/1     Completed   0          53m
kafka-producer-job-8c2tj   0/1     Completed   0          64m
kafka-producer-job-8c2tj   0/1     Completed   0          64m
kafka-producer-job-r6b5l   0/1     Pending     0          0s
kafka-producer-job-r6b5l   0/1     Pending     0          0s
kafka-producer-job-r6b5l   0/1     ContainerCreating   0          0s
kafka-producer-job-r6b5l   1/1     Running             0          2s
golang-kafka-consumer-86b75dcdf4-cnckr   0/1     Pending             0          0s
golang-kafka-consumer-86b75dcdf4-cnckr   0/1     Pending             0          0s
golang-kafka-consumer-86b75dcdf4-cnckr   0/1     ContainerCreating   0          0s
golang-kafka-consumer-86b75dcdf4-cnckr   1/1     Running             0          2s
golang-kafka-consumer-86b75dcdf4-4kbgl   0/1     Pending             0          0s
golang-kafka-consumer-86b75dcdf4-4kbgl   0/1     Pending             0          0s
golang-kafka-consumer-86b75dcdf4-4kbgl   0/1     ContainerCreating   0          0s
golang-kafka-consumer-86b75dcdf4-4kbgl   1/1     Running             0          2s
kafka-producer-job-r6b5l                 0/1     Completed           0          62s
kafka-producer-job-r6b5l                 0/1     Completed           0          63s
kafka-producer-job-r6b5l                 0/1     Completed           0          64s
golang-kafka-consumer-86b75dcdf4-4kbgl   1/1     Terminating         0          4m27s
golang-kafka-consumer-86b75dcdf4-cnckr   1/1     Terminating         0          4m30s
golang-kafka-consumer-86b75dcdf4-4kbgl   0/1     Error               0          4m28s
golang-kafka-consumer-86b75dcdf4-cnckr   0/1     Error               0          4m31s
golang-kafka-consumer-86b75dcdf4-cnckr   0/1     Error               0          4m31s
golang-kafka-consumer-86b75dcdf4-cnckr   0/1     Error               0          4m31s
golang-kafka-consumer-86b75dcdf4-4kbgl   0/1     Error               0          4m28s
golang-kafka-consumer-86b75dcdf4-4kbgl   0/1     Error               0          4m28s
```

``` bash
kubectl get pods
NAME                       READY   STATUS      RESTARTS   AGE
kafka-producer-job-8c2tj   0/1     Completed   0          58m
```

```bash
 kubectl get hpa -n default -w
NAME                                   REFERENCE                          TARGETS              MINPODS   MAXPODS   REPLICAS   AGE
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   <unknown>/10 (avg)   1         5         0          103m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   11/10 (avg)          1         5         1          117m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   5/10 (avg)           1         5         2          117m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   8500m/10 (avg)       1         5         2          120m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   7/10 (avg)           1         5         2          120m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   5500m/10 (avg)       1         5         2          121m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   4/10 (avg)           1         5         2          121m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   2500m/10 (avg)       1         5         2          121m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   1/10 (avg)           1         5         2          121m
keda-hpa-kafka-consumer-scaledobject   Deployment/golang-kafka-consumer   <unknown>/10 (avg)   1         5         0          122m
```
