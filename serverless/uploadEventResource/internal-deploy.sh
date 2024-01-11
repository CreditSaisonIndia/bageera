#!/usr/bin/env bash

echo "Deploying Partner Workflow withdrawl Lambdas"

AWS_REGION=$1
BUCKET_NAME="$2"
ENV=$3
AWS_PROFILE=$4

STACK_NAME=onion-partner-workflow-withdrawl


echo "Stack Name: $STACK_NAME"


build_lambda() {
  (
    echo "-----Packaging $1 lambda-----"
    cd "$1" || exit
    sam build --use-container
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

sam package \
  --template-file master.yaml \
  --output-template-file temp-template.template \
  --s3-bucket $BUCKET_NAME \
  --region $AWS_REGION \
  --profile $AWS_PROFILE

sam deploy \
    --stack-name $STACK_NAME \
    --template-file temp-template.template \
    --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND \
    --s3-bucket $BUCKET_NAME \
    --region $AWS_REGION \
    --profile $AWS_PROFILE \
    --parameter-overrides \
    EnvironmentType="${ENV}"

# Clean Up
rm -rf */package.yaml
rm -rf */.aws-sam
rm -f temp-template.template
rm -rf .env