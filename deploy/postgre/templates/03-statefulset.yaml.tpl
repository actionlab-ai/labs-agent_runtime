apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: __APP_NAME__
  namespace: __NAMESPACE__
  labels:
    app.kubernetes.io/name: __APP_NAME__
    app.kubernetes.io/component: database
    app.kubernetes.io/managed-by: manual
spec:
  serviceName: __APP_NAME__-headless
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: __APP_NAME__
      app.kubernetes.io/component: database
  template:
    metadata:
      labels:
        app.kubernetes.io/name: __APP_NAME__
        app.kubernetes.io/component: database
    spec:
      terminationGracePeriodSeconds: 60
      containers:
        - name: postgres
          image: __IMAGE__
          imagePullPolicy: IfNotPresent
          ports:
            - name: postgres
              containerPort: 5432
              protocol: TCP
          env:
            - name: POSTGRES_DB
              valueFrom:
                secretKeyRef:
                  name: __APP_NAME__-secret
                  key: POSTGRES_DB
            - name: POSTGRES_USER
              valueFrom:
                secretKeyRef:
                  name: __APP_NAME__-secret
                  key: POSTGRES_USER
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: __APP_NAME__-secret
                  key: POSTGRES_PASSWORD
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata
            - name: TZ
              value: __TZ__
          args:
            - "-c"
            - "max_connections=200"
            - "-c"
            - "shared_buffers=256MB"
            - "-c"
            - "log_min_duration_statement=1000"
          readinessProbe:
            exec:
              command:
                - /bin/sh
                - -c
                - pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" -h 127.0.0.1 -p 5432
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 6
          livenessProbe:
            exec:
              command:
                - /bin/sh
                - -c
                - pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" -h 127.0.0.1 -p 5432
            initialDelaySeconds: 30
            periodSeconds: 20
            timeoutSeconds: 5
            failureThreshold: 6
          resources:
            requests:
              cpu: __CPU_REQUEST__
              memory: __MEMORY_REQUEST__
            limits:
              cpu: __CPU_LIMIT__
              memory: __MEMORY_LIMIT__
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: data
        labels:
          app.kubernetes.io/name: __APP_NAME__
          app.kubernetes.io/component: database
      spec:
        accessModes:
          - __ACCESS_MODE__
        storageClassName: "__STORAGE_CLASS__"
        resources:
          requests:
            storage: __STORAGE_SIZE__
