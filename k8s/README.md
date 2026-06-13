# Kit de despliegue — NATS JetStream en Kubernetes (OrbStack)

Manifests y comandos para **volver a levantar** el laboratorio de NATS JetStream
en un cluster local (probado en **OrbStack**, 1 nodo). Pensado para encender el
cluster cada cierto tiempo y desplegar el lab en minutos.

```
k8s/
├── nats-values.yaml                 # values de NATS (ajustados a 1 nodo)
└── monitoring/                      # OPCIONAL: observabilidad
    ├── kube-prometheus-stack-values.yaml
    ├── grafana-ingressroute.yml     # IngressRoute Traefik + redirect https
    ├── prometheus-ingressroute.yml
    └── nats-dashboard.json          # dashboard de NATS para Grafana
```

## Requisitos

- Cluster activo (`orb start` si usas OrbStack) y `kubectl` apuntando a él.
- `helm` instalado.
- Repos de Helm:
  ```bash
  helm repo add nats https://nats-io.github.io/k8s/helm/charts/
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
  helm repo update
  ```

---

## 1) Desplegar NATS (lo esencial)

```bash
helm install nats nats/nats \
  --namespace nats-system --create-namespace \
  -f nats-values.yaml
```

### Verificar

```bash
kubectl get pods -n nats-system          # 3 réplicas Running + nats-box
kubectl get pvc  -n nats-system          # 3 PVCs Bound (local-path)

# Salud del cluster JetStream (meta leader + quórum)
kubectl exec -n nats-system nats-0 -c nats -- wget -qO- http://localhost:8222/jsz \
  | python3 -c "import sys,json;d=json.load(sys.stdin);print('meta leader:',d['meta_cluster']['leader'],'| size:',d['meta_cluster']['cluster_size'])"

# Prueba rápida: crear un stream replicado y publicar
kubectl exec -n nats-system deploy/nats-box -- \
  nats --server nats://nats:4222 stream add LAB --subjects 'lab.>' --replicas 3 --storage file --defaults
kubectl exec -n nats-system deploy/nats-box -- \
  nats --server nats://nats:4222 pub lab.test "hola"
```

### Conectarse desde el Mac (host)

OrbStack enruta la red del cluster, así que **no necesitas `port-forward`**:

```bash
# DNS del Service (estable)
export NATS_URL=nats://nats.nats-system.svc.cluster.local:4222
```

> En otros entornos sin esa red enrutada, usa:
> `kubectl -n nats-system port-forward svc/nats 4222:4222`

---

## 2) Observabilidad (OPCIONAL)

> Requiere un Ingress. En este lab **Traefik** era el ingress/LoadBalancer.
> Si no lo tienes, instala el stack igualmente y usa `port-forward` para Grafana.

```bash
# Stack de Prometheus + Grafana (ligero, ajustado a 1 nodo)
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace \
  -f monitoring/kube-prometheus-stack-values.yaml

# Exponer Grafana/Prometheus vía Traefik (https://*.k8s.orb.local)
kubectl apply -f monitoring/grafana-ingressroute.yml \
               -f monitoring/prometheus-ingressroute.yml

# Importar el dashboard de NATS (el sidecar de Grafana lo autocarga)
kubectl create configmap nats-dashboard -n monitoring \
  --from-file=nats-dashboard.json=monitoring/nats-dashboard.json
kubectl label configmap nats-dashboard -n monitoring grafana_dashboard=1 --overwrite
```

Accesos (cert self-signed de Traefik → acepta el aviso del navegador):
- Grafana → https://grafana.k8s.orb.local (admin / admin)
- Prometheus → https://prometheus.k8s.orb.local

**Sin Traefik**, accede con port-forward:
```bash
kubectl -n monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80
```

---

## 3) Limpiar (teardown)

```bash
# NATS (incluye PVCs → libera disco)
helm uninstall nats -n nats-system && kubectl delete ns nats-system

# Monitoreo
helm uninstall kube-prometheus-stack -n monitoring && kubectl delete ns monitoring
kubectl get crd -o name | grep monitoring.coreos.com | xargs kubectl delete
```

---

## Notas del entorno

- **1 nodo**: las 3 réplicas caen en el mismo host. Es HA "lógica" para practicar
  replicación/RAFT, no tolerancia a fallos de hardware real.
- **StorageClass**: se usa `local-path` (la default de OrbStack). No existe `fast-ssd`.
- **Recursos**: los values están ajustados para un nodo de ~8 Gi RAM / 10 CPU.
- ⚠️ **Traefik vivía en el namespace `argocd`** (instalado ahí por error). Es el
  único ingress/LoadBalancer del cluster: **no borres ese namespace** o pierdes el ingress.
