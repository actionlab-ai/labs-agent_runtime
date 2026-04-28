# PostgreSQL 16.13 Kubernetes 部署工程

本工程用于在 Kubernetes 中部署单实例 PostgreSQL，镜像使用：

```text
swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/postgres:16.13-alpine3.23
```

部署形态：

- `StatefulSet`：保证 Pod 名称稳定，例如 `postgres-0`
- `PVC`：持久化 PostgreSQL 数据目录
- `Secret`：保存数据库用户名、密码、库名
- `ClusterIP Service`：集群内部访问 PostgreSQL
- `Headless Service`：给 StatefulSet 使用
- 可选 `NodePort Service`：需要集群外访问时再开启

> 注意：这是单实例 PostgreSQL，不是高可用集群。不要直接把 `replicas` 改成 2 或 3，否则多个实例会抢同一套逻辑角色，数据一致性不可控。需要高可用请使用 CloudNativePG、Zalando Postgres Operator、Patroni 等方案。

---

## 1. 目录结构

```text
postgres-k8s-deploy/
├── README.md
├── env.example
├── install.sh
├── uninstall.sh
└── templates/
    ├── 00-namespace.yaml.tpl
    ├── 01-secret.yaml.tpl
    ├── 02-service.yaml.tpl
    ├── 03-statefulset.yaml.tpl
    └── 04-service-nodeport.yaml.tpl
```

执行 `install.sh` 后，会在 `rendered/` 目录生成最终 YAML，然后执行 `kubectl apply`。

---

## 2. 默认部署信息

| 配置项 | 默认值 |
|---|---|
| Namespace | `codefly` |
| 应用名 | `postgres` |
| 镜像 | `swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/postgres:16.13-alpine3.23` |
| Service 名称 | `postgres` |
| Headless Service 名称 | `postgres-headless` |
| Pod 名称 | `postgres-0` |
| 数据库端口 | `5432` |
| 默认数据库 | `appdb` |
| 默认用户 | `postgres` |
| 默认密码 | 通过 `POSTGRES_PASSWORD` 显式指定 |
| 默认 StorageClass | `nfs` |
| 默认容量 | `20Gi` |
| 默认访问模式 | `ReadWriteOnce` |

集群内连接地址：

```text
postgres.aict.svc.cluster.local:5432
```

同命名空间内可以简写为：

```text
postgres:5432
```

---

## 3. 快速部署

### 3.1 修改变量

复制环境变量示例：

```bash
cp env.example .env
vi .env
```

示例：

```bash
export NAMESPACE=aict
export APP_NAME=postgres
export STORAGE_CLASS=nfs
export STORAGE_SIZE=50Gi
export POSTGRES_DB=codefly
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD='change-me'
```

### 3.2 执行部署

```bash
source .env
chmod +x install.sh uninstall.sh
./install.sh
```

查看状态：

```bash
kubectl get pod,svc,pvc,secret -n aict -l app.kubernetes.io/name=postgres
kubectl rollout status sts/postgres -n aict
```

查看日志：

```bash
kubectl logs -n aict sts/postgres -f
```

---

## 4. 常用部署命令

### 4.1 使用默认配置部署

```bash
chmod +x install.sh
./install.sh
```

### 4.2 指定 StorageClass 和容量

```bash
export STORAGE_CLASS=nfs
export STORAGE_SIZE=100Gi
./install.sh
```

### 4.3 指定数据库、用户、密码

```bash
export POSTGRES_DB=oneberry
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD='YourStrongPassword@123'
./install.sh
```

### 4.4 指定命名空间

```bash
export NAMESPACE=aict
./install.sh
```

### 4.5 开启 NodePort 暴露

默认只创建 `ClusterIP`，更安全。需要集群外访问时再开启：

```bash
export ENABLE_NODEPORT=true
export NODE_PORT=30432
./install.sh
```

集群外访问地址：

```text
任意NodeIP:30432
```

---

## 5. 连接 PostgreSQL

### 5.1 在集群内临时启动 psql 客户端

```bash
kubectl run psql-client \
  -n aict \
  --rm -it \
  --restart=Never \
  --image=swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/postgres:16.13-alpine3.23 \
  --env PGPASSWORD='<your-postgres-password>' \
  -- psql -h postgres -p 5432 -U postgres -d appdb
```

连接成功后可以执行：

```sql
select version();
\l
\dt
```

### 5.2 进入 PostgreSQL Pod 本机连接

```bash
kubectl exec -it -n aict postgres-0 -- bash
psql -U postgres -d appdb
```

### 5.3 获取 Secret 中的密码

```bash
kubectl get secret -n aict postgres-secret \
  -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d; echo
```

---

## 6. 数据目录说明

容器内数据目录：

```text
/var/lib/postgresql/data/pgdata
```

这里显式设置了：

```text
PGDATA=/var/lib/postgresql/data/pgdata
```

原因是很多存储卷挂载到 `/var/lib/postgresql/data` 后会出现 `lost+found` 目录，PostgreSQL 初始化时可能认为数据目录不为空。使用子目录 `pgdata` 可以规避这个问题。

---

## 7. 变量说明

| 变量 | 默认值 | 说明 |
|---|---:|---|
| `NAMESPACE` | `aict` | 部署命名空间 |
| `APP_NAME` | `postgres` | 应用名、Service 名、StatefulSet 名 |
| `IMAGE` | 华为云 SWR postgres 镜像 | PostgreSQL 镜像 |
| `STORAGE_CLASS` | `nfs` | PVC 使用的 StorageClass。为空时使用集群默认 StorageClass |
| `STORAGE_SIZE` | `20Gi` | PVC 容量 |
| `ACCESS_MODE` | `ReadWriteOnce` | PVC 访问模式 |
| `POSTGRES_DB` | `appdb` | 初始化数据库名 |
| `POSTGRES_USER` | `postgres` | 初始化用户名 |
| `POSTGRES_PASSWORD` | 必填 | 初始化用户密码 |
| `TZ` | `Asia/Shanghai` | 容器时区 |
| `CPU_REQUEST` | `100m` | CPU request |
| `MEMORY_REQUEST` | `256Mi` | Memory request |
| `CPU_LIMIT` | `2` | CPU limit |
| `MEMORY_LIMIT` | `2Gi` | Memory limit |
| `ENABLE_NODEPORT` | `false` | 是否创建 NodePort Service |
| `NODE_PORT` | `30432` | NodePort 端口 |

---

## 8. 修改密码的注意事项

PostgreSQL 官方镜像的 `POSTGRES_PASSWORD` 只在第一次初始化空数据目录时生效。

如果 PVC 已经存在，后续只改 Secret 或环境变量，不会自动修改数据库里已有用户的密码。

正确修改方式：

```bash
kubectl exec -it -n aict postgres-0 -- psql -U postgres -d appdb
```

进入后执行：

```sql
ALTER USER postgres WITH PASSWORD 'NewStrongPassword@123';
```

然后再同步更新 Kubernetes Secret，避免后续运维混乱。

---

## 9. 卸载

只删除工作负载、Service、Secret，不删除 PVC：

```bash
./uninstall.sh
```

连 PVC 一起删除：

```bash
export DELETE_PVC=true
./uninstall.sh
```

删除 PVC 会删除数据库数据，请谨慎操作。

---

## 10. 常见问题

### 10.1 Pod 卡在 Pending

查看 PVC 是否绑定成功：

```bash
kubectl get pvc -n aict
kubectl describe pvc -n aict data-postgres-0
```

重点检查：

- `STORAGE_CLASS` 是否存在
- StorageClass 是否支持动态创建 PV
- NFS provisioner 是否正常

查看 StorageClass：

```bash
kubectl get sc
```

### 10.2 PostgreSQL 初始化失败

查看日志：

```bash
kubectl logs -n aict postgres-0
```

常见原因：

- PVC 没有写权限
- NFS root squash 导致 chown 失败
- 密码为空或环境变量写错
- 旧 PVC 中已有残留数据

### 10.3 应用连接不上

检查 Service：

```bash
kubectl get svc -n aict postgres
kubectl get endpoints -n aict postgres
```

检查 Pod readiness：

```bash
kubectl get pod -n aict postgres-0
kubectl describe pod -n aict postgres-0
```

集群内应用连接配置一般为：

```text
host=postgres.aict.svc.cluster.local
port=5432
database=appdb
username=postgres
password=<your-postgres-password>
```

同命名空间应用可使用：

```text
host=postgres
port=5432
```

---

## 11. 生产建议

这个工程适合：

- 测试环境
- 开发环境
- 内部小型系统
- 对高可用要求不高的业务组件

生产环境建议补充：

1. 定时备份：`pg_dump`、`pg_basebackup` 或备份到 MinIO/S3。
2. 监控告警：Prometheus + postgres_exporter。
3. 高可用：使用 PostgreSQL Operator 或 Patroni。
4. 网络策略：限制只有指定应用命名空间可以访问 5432。
5. 密码管理：接入 Vault、ExternalSecrets 或 SealedSecrets。
6. 资源调优：根据实际数据量设置 shared_buffers、work_mem、max_connections。
