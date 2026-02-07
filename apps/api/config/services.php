<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Third Party Services
    |--------------------------------------------------------------------------
    |
    | This file is for storing the credentials for third party services such
    | as Mailgun, Postmark, AWS and more. This file provides the de facto
    | location for this type of information, allowing packages to have
    | a conventional file to locate the various service credentials.
    |
    */

    'postmark' => [
        'key' => env('POSTMARK_API_KEY'),
    ],

    'resend' => [
        'key' => env('RESEND_API_KEY'),
    ],

    'ses' => [
        'key' => env('AWS_ACCESS_KEY_ID'),
        'secret' => env('AWS_SECRET_ACCESS_KEY'),
        'region' => env('AWS_DEFAULT_REGION', 'us-east-1'),
    ],

    'slack' => [
        'notifications' => [
            'bot_user_oauth_token' => env('SLACK_BOT_USER_OAUTH_TOKEN'),
            'channel' => env('SLACK_BOT_USER_DEFAULT_CHANNEL'),
        ],
    ],

    'engine' => [
        'api_url' => env('ENGINE_API_URL', 'http://linkflow-api:8000'),
        'secret' => env('LINKFLOW_ENGINE_SECRET', env('LINKFLOW_SECRET')),
        'callback_ttl' => (int) env('ENGINE_CALLBACK_TTL', 300),
        'partition_count' => (int) env('ENGINE_PARTITION_COUNT', 16),
        'stream_maxlen' => (int) env('ENGINE_STREAM_MAXLEN', 100000),
        'send_sensitive_context' => (bool) env('ENGINE_SEND_SENSITIVE_CONTEXT', false),
    ],

];
