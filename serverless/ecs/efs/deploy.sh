#!/usr/bin/env bash


PS3='Please enter the environment (enter corresponding number): '
options=("dev" "qa" "qa2" "uat" "int" "production")
select opt in "${options[@]}"
do
  case $opt in
    "dev"|"qa"|"uat"|"int")
      AWS_REGION=us-east-1
      BUCKET_NAME="cfn-templates-v2-$opt"
      ENV=$opt
      AWS_PROFILE=Development-Technology-Developer-971709774307
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

sh deploy-internal.sh "$ENV" $AWS_REGION $BUCKET_NAME $AWS_PROFILE

