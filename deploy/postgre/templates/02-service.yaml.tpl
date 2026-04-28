apiVersion: v1
kind: Service
metadata:
  name: __APP_NAME__-headless
  namespace: __NAMESPACE__
  labels:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
    app.kubernetes.io/managed-by: manual
spec:
  clusterIP: None
  publishNotReadyAddresses: true
  selector:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
  ports:
    - name: postgres
      port: 5432
      targetPort: postgres
      protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: __APP_NAME__
  namespace: __NAMESPACE__
  labels:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
    app.kubernetes.io/managed-by: manual
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
  ports:
    - name: postgres
      port: 5432
      targetPort: postgres
      protocol: TCP
