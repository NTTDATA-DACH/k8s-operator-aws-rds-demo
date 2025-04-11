#!/bin/zsh

aws iam create-policy --no-verify-ssl --policy-name rds-operator-policy --policy-document file://aws/rds-operator-policy.json

eksctl utils associate-iam-oidc-provider --cluster aboczek-test-cluster-manu3 --approve

eksctl create iamserviceaccount --name aws-rds-operator --namespace k8s-operator-aws-rds-demo-system --cluster aboczek-test-cluster-manu3 --attach-policy-arn arn:aws:iam::063010046250:policy/rds-operator-policy --approve