apiVersion: v1
kind: Service
metadata:
  name: __APP_NAME__-nodeport
  namespace: __NAMESPACE__
  labels:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
    app.kubernetes.io/managed-by: manual
spec:
  type: NodePort
  selector:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
  ports:
    - name: postgres
      port: 5432
      targetPort: postgres
      nodePort: __NODE_PORT__
      protocol: TCP
