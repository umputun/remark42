---
title: Deploy with Kubernetes
---

## Using Helm

Use [remark42 Helm chart](https://github.com/groundhog2k/helm-charts/tree/master/charts/remark42).

## Without Helm

Here's the sample manifest for running remark42 on Hetzner Cloud:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: remark42
  namespace: remark42
  labels:
    app: remark42
spec:
  replicas: 1
  selector:
    matchLabels:
      app: remark42
  strategy:
    type: Recreate
  template:
    metadata:
      namespace: remark42
      labels:
        app: remark42
    spec:
      containers:
        - name: remark42
          image: umputun/remark42:v1.8.1
          ports:
            # http:
            - containerPort: 8080
          env:
            - name: REMARK_URL
              value: "https://comments.mysite.com/"
            - name: "SITE"
              value: "mysite.com"
            - name: SECRET
              valueFrom:
                secretKeyRef:
                  name: remark42
                  key: SECRET
            - name: STORE_BOLT_PATH
              value: "/srv/var/db"
            - name: BACKUP_PATH
              value: "/srv/var/backup"
            - name: AUTH_GOOGLE_CID
              valueFrom:
                secretKeyRef:
                  name: remark42
                  key: AUTH_GOOGLE_CID
            - name: AUTH_GOOGLE_CSEC
              valueFrom:
                secretKeyRef:
                  name: remark42
                  key: AUTH_GOOGLE_CSEC
            - name: AUTH_GITHUB_CID
              valueFrom:
                secretKeyRef:
                  name: remark42
                  key: AUTH_GITHUB_CID
            - name: AUTH_GITHUB_CSEC
              valueFrom:
                secretKeyRef:
                  name: remark42
                  key: AUTH_GITHUB_CSEC
            - name: ADMIN_SHARED_ID
              value: "google_b182b5daa0004104b348d9bde762b1880ed9d98d"
            - name: TIME_ZONE
              value: "Europe/Dublin"
          volumeMounts:
            - name: srvvar
              mountPath: /srv/var
          securityContext:
            readOnlyRootFilesystem: false
          resources:
            requests:
              cpu: "100m"
              memory: "25Mi"
            limits:
              cpu: "1"
              memory: "1Gi"
      securityContext:
        # Has its own root privilege drop. Can't do runAsUser / runAsGroup.
      volumes:
        - name: srvvar
          persistentVolumeClaim:
            claimName: remark42
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: remark42
  namespace: remark42
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: hcloud-volumes
---
apiVersion: v1
kind: Service
metadata:
  name: remark42-web
  namespace: remark42
spec:
  selector:
    app: remark42
  ports:
    - name: http
      protocol: TCP
      port: 8080
      targetPort: 8080
---
# TODO: switch to networking.k8s.io/v1
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: remark42-ingress
  namespace: remark42
  annotations:
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  tls:
    - hosts:
        - comments.mysite.com
      secretName: comments-tls
  rules:
    - host: "comments.mysite.com"
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              serviceName: remark42-web
              servicePort: 8080
```

Change `storageClassName` if you run on top of different cloud / bare metal.

This example assumes there is Nginx Ingress with a cert-manager already set up.
Adjust if you use different Ingress.

In addition you'd need to define secrets, e.g.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: remark42
  namespace: remark42
stringData:
  SECRET: <changeme>
  AUTH_GOOGLE_CID: <changeme>.apps.googleusercontent.com
  AUTH_GOOGLE_CSEC: <changeme>
  AUTH_GITHUB_CID: <changeme>
  AUTH_GITHUB_CSEC: <changeme>
```

Some more information (and comments!) may be found
[here](https://www.rusinov.ie/en/posts/2021/this-website-has-remark42-comments-now/).
