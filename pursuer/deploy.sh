# !/usr/bin/env bash

declare -a ENV_ARRAY
ENV_ARRAY=('dev' 'qa' 'qa2' 'uat' 'int' 'production')

getIndex() {
  for i in "${!ENV_ARRAY[@]}"; do
    if [[ "${ENV_ARRAY[$i]}" == "${1}" ]]; then
      return "${i}"
    fi
  done
  return -1
}

AWS_PROFILE=default

PS3='Please enter the environment (enter corresponding number): '
options=("dev" "qa" "qa2" "uat" "int" "production")
select opt in "${options[@]}"
do
  case $opt in
    "dev"|"qa"|"uat"|"int")
      AWS_REGION=us-east-1
      AWS_PROFILE=Development-Technology-Developer-971709774307
      break
      ;;
    "qa2")
      AWS_REGION=eu-west-1
      break
      ;;
    "production")
      AWS_REGION=ap-south-1
      break
      ;;
    *) echo "invalid option";;
    esac
done

echo "Setting region to $AWS_REGION"

#Fetch the account ID
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text --profile $AWS_PROFILE)

#Build your Docker image
#docker build --platform=linux/amd64 -f Containers/Core/Dockerfile -t core .

ContainerPath=pursuer

echo "-----Packaging $ContainerPath image-----"
ContainerImage=$(echo "bageera-$ContainerPath-repo" | tr '[:upper:]' '[:lower:]')
echo $ContainerImage
aws ecr get-login-password --region $AWS_REGION --profile $AWS_PROFILE | docker login --username AWS --password-stdin $ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com
docker build --platform=linux/amd64 -f Dockerfile -t $ContainerImage .
docker tag $ContainerImage:latest $ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ContainerImage:latest
docker push $ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ContainerImage:latest


