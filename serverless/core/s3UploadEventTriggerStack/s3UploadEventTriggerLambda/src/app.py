import json
import boto3
import logging
import os

logger = logging.getLogger()
logger.setLevel(logging.DEBUG)

SNS_TOPIC_ARN = os.getenv('SNS_TOPIC_ARN')

def lambda_handler(event, context):
    # Extract necessary information from the S3 event

    logger.info("Event from S3 : %s", json.dumps(event))    
    
    s3_event = event['Records'][0]['s3']
    bucket_name = s3_event['bucket']['name']
    object_key = s3_event['object']['key']
    
    logger.info("BUCKET NAME : %s", bucket_name)
    logger.info("OBJECT KEY : %s", object_key)
    file_name = os.path.basename(object_key)
    path_elements = object_key.split("/")

    # Create an SNS client
    sns_client = boto3.client('sns')

    # Create a message to publish to the SNS topic
    message = {
        'event': 'S3_UPLOAD',
        'bucketName': bucket_name,
        'objectKey': object_key,
        'lpc':path_elements[2],
        'fileName':file_name,
        'environment':"dev",
        'requestQueueUrl':"https://sqs.us-east-1.amazonaws.com/971709774307/pq-job-queue-us-east-1"
        # 'dbUserName':"masteruser",
        # 'dbPassword':"szkeUq2DbgHAUnW",
        # 'dbHost':"ksf-cluster-v2-us-east-1.cluster-cpktqakm2slz.us-east-1.rds.amazonaws.com",
        # 'dbPort':"5432",
        # 'dbName':"proddb",
        # 'schema':"scarlet",
        # 'efsBasePath':"/app/bageera/temp/data",
        # 'environment':"dev"
    }

    # Publish the message to the SNS topic
    response = sns_client.publish(
        TopicArn=SNS_TOPIC_ARN,
        Message=json.dumps({'default': json.dumps(message)}),
        MessageStructure='json'
    )

    logger.info(f"Message published to SNS: {response}")