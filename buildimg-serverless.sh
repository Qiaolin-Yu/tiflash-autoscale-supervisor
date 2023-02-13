if [[ $# -eq 0 ]] ; then
    echo 'need args!'
    exit 1
fi
#docker build -t bestwoody/supervisor:serverless.v6.4.0 .  && docker push  bestwoody/supervisor:serverless.v6.4.0
# docker build -t 646528577659.dkr.ecr.us-east-2.amazonaws.com/tiflash-autoscale-supervisor:serverless.v6.4.0 .  && docker push  646528577659.dkr.ecr.us-east-2.amazonaws.com/tiflash-autoscale-supervisor:serverless.v6.4.0
#docker buildx build --platform=linux/amd64,linux/arm64 -t 646528577659.dkr.ecr.us-east-2.amazonaws.com/tiflash-autoscale-supervisor:serverless.v6.4.0 .  --push
docker buildx build --platform=linux/amd64,linux/arm64 -t 646528577659.dkr.ecr.us-east-2.amazonaws.com/tiflash-autoscale-supervisor:$1 .  --push