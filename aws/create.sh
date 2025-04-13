#!/bin/zsh

aws iam create-policy --no-verify-ssl --policy-name rds-operator-policy --policy-document file://aws/rds-operator-policy.json

eksctl utils associate-iam-oidc-provider --cluster aboczek-test-cluster-manu3 --approve

eksctl create iamserviceaccount --name aws-rds-operator --namespace k8s-operator-aws-rds-demo-system --cluster aboczek-test-cluster-manu3 --attach-policy-arn arn:aws:iam::063010046250:policy/rds-operator-policy --approve

make generate
make manifests

make docker-build docker-push IMG="063010046250.dkr.ecr.eu-central-1.amazonaws.com/k8s-operator-aws-rds-demo:v0.0.18"

docker login --username AWS -p $(aws ecr get-login-password --no-verify-ssl --region eu-central-1) 063010046250.dkr.ecr.eu-central-1.amazonaws.com
make deploy IMG="063010046250.dkr.ecr.eu-central-1.amazonaws.com/k8s-operator-aws-rds-demo:v0.0.18"

kubectl apply -f config/samples/aws_v1alpha1_awsrdsdemoinstance.yaml