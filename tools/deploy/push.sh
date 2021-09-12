#!/bin/bash -ue
cd "$(dirname $0)"

registry="registry.kleinpa.net"
prefix="battleship"
tag=$(date +"%Y%m%d%H%M")

kubectl -n counter create secret generic kleinpa-registry-cert \
    --from-file=.dockerconfigjson=$HOME/.docker/config.json \
    --type=kubernetes.io/dockerconfigjson

if [ -x "$(command -v docker)" ]; then
    docker build -f client.Dockerfile ../client --tag ${registry}/${prefix}/static:${tag} --tag ${registry}/${prefix}/static:latest
    docker push ${registry}/${prefix}/static:${tag}
else
    echo 'docker is not installed, skipping client build' >&2
fi

bazel build //server:server_image.tar
crane push $(bazel info bazel-bin)/server/server_image.tar ${registry}/${prefix}/server:${tag}

bazel build //deploy:proxy_image.tar
crane push $(bazel info bazel-bin)/deploy/proxy_image.tar ${registry}/${prefix}/proxy:${tag}


kubectl -n=counter set image deployment/battleship-static battleship-static=${registry}/${prefix}/static:${tag}
kubectl -n=counter set image deployment/battleship-proxy battleship-proxy=${registry}/${prefix}/proxy:${tag}
kubectl -n=counter set image statefulset/battleship-server battleship-server=${registry}/${prefix}/server:${tag}
