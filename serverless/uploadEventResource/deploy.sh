#!/usr/bin/env bash

AWS_PROFILE=default

PS3='Please enter the environment (enter corresponding number): '
options=("dev" "qa" "qa2" "uat" "int" "production")
select opt in "${options[@]}"

do
  case $opt in
    "dev"|"qa"|"uat"|"int")
      AWS_REGION=us-east-1
      BUCKET_NAME="cfn-templates-v2-$opt"
      ENV=$opt
      break
      ;;
    "qa2")
      AWS_REGION=eu-west-1
      BUCKET_NAME=cfn-templates-qa2
      ENV=$opt
      break
      ;;
      "production")
      AWS_REGION=ap-south-1
      BUCKET_NAME=cfn-oneaboveall-templates-production
      ENV=$opt
      break
      ;;
    *) echo "invalid option";;
    esac
done

sh internal-deploy.sh "$AWS_REGION" "$BUCKET_NAME" "$ENV" "$AWS_PROFILE"
