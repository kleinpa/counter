#!/bin/bash -ue
cd $(dirname $0)/../..

registry="registry.kleinpa.net"
prefix="counter"
tag=$(date +"%Y%m%d%H%M")

# kubectl -n counter create secret generic kleinpa-registry-cert \
#     --from-file=.dockerconfigjson=$HOME/.docker/config.json \
#     --type=kubernetes.io/dockerconfigjson

docker build . -f tools/deploy/server.Dockerfile --tag ${prefix}/server:${tag}
docker tag ${prefix}/server:${tag} ${registry}/${prefix}/server:${tag}
docker push ${registry}/${prefix}/server:${tag}
echo Pushed ${registry}/${prefix}/server:${tag} >&2
kubectl -n=counter set image statefulset/counter-server counter-server=${registry}/${prefix}/server:${tag}

docker build . -f tools/deploy/static.Dockerfile --tag ${prefix}/static:${tag}
docker tag ${prefix}/static:${tag} ${registry}/${prefix}/static:${tag}
docker push ${registry}/${prefix}/static:${tag}
echo Pushed ${registry}/${prefix}/static:${tag} >&2
kubectl -n=counter set image deployment/counter-static counter-static=${registry}/${prefix}/static:${tag}

docker build . -f tools/deploy/proxy.Dockerfile --tag ${prefix}/proxy:${tag}
docker tag ${prefix}/proxy:${tag} ${registry}/${prefix}/proxy:${tag}
docker push ${registry}/${prefix}/proxy:${tag}
echo Pushed ${registry}/${prefix}/proxy:${tag} >&2
kubectl -n=counter set image deployment/counter-proxy counter-proxy=${registry}/${prefix}/proxy:${tag}
