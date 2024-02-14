import json
import boto3
import logging
import os

logger = logging.getLogger()
logger.setLevel(logging.DEBUG)

UPLOAD_EVENT_SNS_TOPIC_ARN = os.getenv('SNS_TOPIC_ARN')
ALERT_SNS_TOPIC_ARN = os.getenv('ALERT_SNS')
ENV = os.getenv('ENV')
PQ_JOB_QUEUE_URL = os.getenv('PQ_JOB_QUEUE_URL')

def lambda_handler(event, context):
    # Extract necessary information from the S3 event

    logger.info("Event from S3 : %s", json.dumps(event))    
    
    event_details = event['detail']
    request_arameters = event_details['requestParameters']
    bucket_name = request_arameters['bucketName']
    object_key = request_arameters['key']
    
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
        'lpc':path_elements[3],
        'fileName':file_name,
        'environment':ENV,
        'requestQueueUrl':PQ_JOB_QUEUE_URL,
        'alertSnsArn':ALERT_SNS_TOPIC_ARN
    }
    
    logger.info("MESSAGE : %s", json.dumps(message))

    # Publish the message to the SNS topic
    response = sns_client.publish(
        TopicArn=UPLOAD_EVENT_SNS_TOPIC_ARN,
        Message=json.dumps({'default': json.dumps(message)}),
        MessageStructure='json'
    )

    logger.info(f"Message published to SNS: {response}")