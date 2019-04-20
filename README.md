# configmap-reload-operator

## Run a nginx Demo

### Build, install and run ConfigMap Reload Operator
```
mkdir -p $GOPATH/src/github.com/hackerthon2019
cd $GOPATH/src/github.com/hackerthon2019
git clone https://github.com/hackerthon2019/configmap-reload-operator
cd configmap-reload-operator
git checkout master
make dep
make install
sudo kubectl create -f deploy/crds/app_v1alpha1_appservice_crd.yaml
```

### Create a Nginx Deployment
```
sudo kubectl create -f deploy/service_account.yaml
sudo kubectl create -f deploy/role.yaml
sudo kubectl create -f deploy/role_binding.yaml
sudo kubectl apply -f deploy/nginx_deployment.yaml
```
### Get the WebDisplay
```
APPIP=`sudo kubectl get svc/nginx -o jsonpath='{.spec.clusterIP}'`
curl $APPIP
```

This will show an output as follow:
```
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p><em>This is wukong, I can change.(我是悟空，我会72变)</em></p>
</body>
</html>

```
**NOTE** the paragraph of this render is
"This is wukong, I can change.(我是悟空，我会72变)"

### Change the ConfigMap
```
sudo kubectl edit ConfigMap nginx-conf
```
Change root location in data of ConfigMap from:
```
        location / {
            root   /usr/share/nginx/html;
            index  index.html index.htm;
        }
```
to
```
        location / {
            root   /usr/share/nginx/html1;
            index  index.html index.htm;
        }
```
### Delete one
```
PODNAME=`sudo kubectl get pod --selector=app=nginx -o jsonpath="{.items[0].metadata.name}"`
sudo kubectl delete pod $PODNAME
```

### Get The WebDisplay again
```
curl $APPIP
```

This paragraph will change to:
```
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p><em>Change!Change! I'm Mei Houwang.(变变变，我是美猴王)</em></p>
</body>
</html>

```
**NOTE** the paragraph in this render is
"Change!Change! I'm Mei Houwang.(变变变，我是美猴王)"

### Destroy the Nginx Deployment
```
sudo kubectl delete -f deploy/nginx_deployment.yaml
```

### Get more infos of the nginx pod
```
PODNAME=`sudo kubectl get pod --selector=app=nginx -o jsonpath="{.items[0].metadata.name}"`
sudo kubectl exec $PODNAME -it -- /bin/bash
```

### Info
If you want to know more detail operator info, please see operator [user guide][user-guide.md].

[user-guide.md]: https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md
