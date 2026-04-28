apiVersion: v1
kind: Secret
metadata:
  name: __APP_NAME__-secret
  namespace: __NAMESPACE__
  labels:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
    app.kubernetes.io/managed-by: manual
type: Opaque
data:
  POSTGRES_DB: __POSTGRES_DB_B64__
  POSTGRES_USER: __POSTGRES_USER_B64__
  POSTGRES_PASSWORD: __POSTGRES_PASSWORD_B64__
