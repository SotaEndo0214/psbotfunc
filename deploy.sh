#!/bin/bash -xe

gcloud functions deploy pokemonsleepbot \
    --gen2 \
    --runtime=go121 \
    --region asia-northeast2 \
    --source . \
    --entry-point=PokemonSleepFoods \
    --trigger-http \
    --allow-unauthenticated \
    --max-instances=5 \
    --cpu=1 \
    --memory=1Gi \
    --set-env-vars=SLACK_AUTH_TOKEN=$PUBLIC_SLACK_AUTH_TOKEN \
    --set-env-vars=SLACK_SIGNING_SECRETS=$PUBLIC_SLACK_SIGNING_SECRETS \
    --set-env-vars=POKEMONSLEEP_FOODS_JSON_PATH=/workspace/serverless_function_source_code/data/foods.json \
    --set-env-vars=POKEMONSLEEP_COOKS_JSON_PATH=/workspace/serverless_function_source_code/data/cooks.json \
