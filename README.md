# aws-dashboard

Simple AWS dashboard (work in progress).

## Configuration File

aws-dashboard.yaml

    general:
        defaultRegion: us-east-1

    billing:
        bucketName: my-billing-bucket

    slack:
        incomingWebhookUrl: https://hooks.slack.com/services/...
        channel: "#my-slack-channel"
        username: AWS reporter
        iconUrl: https://upload.wikimedia.org/wikipedia/commons/thumb/5/5c/AWS_Simple_Icons_AWS_Cloud.svg/500px-AWS_Simple_Icons_AWS_Cloud.svg.png
