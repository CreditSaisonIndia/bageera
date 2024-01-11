#!/usr/bin/env bash

ENV=$1
STACK_NAME=bageera-consumer
AWS_REGION=$2
BUCKET_NAME=$3
AWS_PROFILE=$4

echo "Deploying $STACK_NAME core template on $1"

sam package --template-file master.yaml \
--output-template-file temp-template.template \
--s3-bucket "$BUCKET_NAME" \
--region "$AWS_REGION" \
--profile "$AWS_PROFILE"

sam deploy --stack-name $STACK_NAME --template-file temp-template.template \
--capabilities CAPABILITY_AUTO_EXPAND CAPABILITY_NAMED_IAM \
--s3-bucket "$BUCKET_NAME" \
--region "$AWS_REGION" \
--profile "$AWS_PROFILE" \
--parameter-overrides EnvironmentType="$ENV"

rm -f temp-template.template

echo "$STACK_NAME template deployed successfully"
