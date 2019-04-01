# lightning-monkey
Lightning monkey is the key solution for automatic deploying &amp; managing Kubernetes clusters.




# How to test it
```shell
#first of all, you have to start up a mongodb instance.
docker run -p 27017:27017 -d --name db \
    -e MONGO_INITDB_ROOT_USERNAME=root \
    -e MONGO_INITDB_ROOT_PASSWORD=root \
    mongo:3.6.10
    
#then, run API server.

docker run -itd \
    -e "BACKEND_STORAGE_ARGS=DRIVER_CONNECTION_STR=mongodb://root:root@127.0.0.1:27017/admin" \
    lm-apiserver:v1.0
    

#last step, run agent.
docker run -itd --restart=always --net=host \
    -v /etc:/etc \
    -v /var/run:/var/run \
    -v /var/lib:/var/lib \
    --entrypoint=/opt/lm-agent \
    repository.gridsum.com:8443/library/lightning_monkey_agent:v0.1 --server=http://IP:PORT --address=10.202.40.100 --metadata=sdfkjsfd83jf73hjfd873hnf --cluster=5c92ffa1d240110001894e09 --etcd --master --cert-dir=/etc/kubernetes/pki
```
