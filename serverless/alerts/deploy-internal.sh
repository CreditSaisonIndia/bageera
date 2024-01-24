#!/usr/bin/env bash

ENV=$1
STACK_NAME=bageera-alerts
AWS_REGION=$2
BUCKET_NAME=$3
AWS_PROFILE=$4

echo "Deploying $STACK_NAME core template on $1"

build_lambda() {
  (
    echo "-----Packaging $1 lambda-----"
    cd "$1" || exit
    sam build
    sam package \
      --output-template-file package.yaml \
      --s3-bucket "${BUCKET_NAME}" \
      --region "$AWS_REGION" \
      --profile $AWS_PROFILE
  )
}


for dir in */; do
      if [ -d "$dir" ]; then
         build_lambda "$dir" &
      fi
done

wait

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

# Clean Up
rm -rf */package.yaml
rm -rf */.aws-sam
rm -f temp-template.template
rm -rf .env

echo "$STACK_NAME template deployed successfully"
