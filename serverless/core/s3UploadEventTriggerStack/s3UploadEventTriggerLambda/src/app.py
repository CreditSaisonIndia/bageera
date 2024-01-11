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

    # Create an SNS client
    sns_client = boto3.client('sns')

    # Create a message to publish to the SNS topic
    message = {
        'event': 'S3_UPLOAD',
        'bucket': bucket_name,
        'object_key': object_key
    }

    # Publish the message to the SNS topic
    response = sns_client.publish(
        TopicArn=SNS_TOPIC_ARN,
        Message=json.dumps({'default': json.dumps(message)}),
        MessageStructure='json'
    )

    logger.info(f"Message published to SNS: {response}")