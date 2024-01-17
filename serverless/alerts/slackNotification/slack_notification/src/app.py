import json
import logging
import os
import requests
import copy

# Initialize logging
logger = logging.getLogger()
logger.setLevel(logging.DEBUG)

PQ_WHITELISTING_ALERTS_SLACK_WEBHOOK_URL = os.getenv('PQ_WHITELISTING_ALERTS_SLACK_WEBHOOK_URL')
NOTIF_FIELD_MAPPING = {
    "lpc" : "LPC",
    "fileName":"File Name",
    "status":"Status",
    "message":"Message",
    "chunkFileName":"Chunked File Name",
    "s3Url":"S3 URL",
    "actionTo":"Action to"
}
BLOCK_LIMIT = 8
PLAIN_TEXT_FIELD_JSON = { "type": "plain_text", "text": "{}"}
MRKDWN_FIELD_JSON = { "type": "mrkdwn", "text": "*{}*" }
NEW_BLOCK = { "type": "section", "fields": [] }

def lambda_handler(event, context):    
    logger.info("Message from SNS: %s", json.dumps(event))

    # Extract bucket and object key
    message = json.loads(event["Records"][0]["Sns"]["Message"])

    with open('notif.json') as notif:
        data = json.load(notif)
    post_to_slack(data, message)
    logger.info("message")



def post_to_slack(data, message):
    webhook_url = PQ_WHITELISTING_ALERTS_SLACK_WEBHOOK_URL
    
    logger.info("webhook url - " + webhook_url)
    block_index = 0
    for key, val in message.items():
        if val !="":
            if len(data["attachments"][0]["blocks"][block_index]["fields"]) + 2 > BLOCK_LIMIT:
                block_index = 2
                data["attachments"][0]["blocks"].append(NEW_BLOCK)
            mrkdwn = copy.deepcopy(MRKDWN_FIELD_JSON)
            mrkdwn["text"] = mrkdwn["text"].format(key)
            data["attachments"][0]["blocks"][block_index]["fields"].append(mrkdwn)
            planTextVal = copy.deepcopy(PLAIN_TEXT_FIELD_JSON)
            planTextVal["text"] = planTextVal["text"].format(val)
            data["attachments"][0]["blocks"][block_index]["fields"].append(planTextVal)

    response = requests.post(
        url=webhook_url, data=json.dumps(data),
        headers={'Content-Type': 'application/json'}
    )
    logger.info("Response - %s", response)
    if response.status_code != 200:
        logger.info("workflow error - %s", data)
        raise Exception("workflow errors cannot be posted to slack.")