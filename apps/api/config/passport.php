<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Passport Guard
    |--------------------------------------------------------------------------
    |
    | Here you may specify which authentication guard Passport will use when
    | authenticating users. This value should correspond with one of your
    | guards that is already present in your "auth" configuration file.
    |
    */

    'guard' => 'web',

    /*
    |--------------------------------------------------------------------------
    | Encryption Keys
    |--------------------------------------------------------------------------
    |
    | Passport uses encryption keys while generating secure access tokens for
    | your application. By default, the keys are stored as local files but
    | can be set via environment variables when that is more convenient.
    |
    */

    'private_key' => env('PASSPORT_PRIVATE_KEY'),

    'public_key' => env('PASSPORT_PUBLIC_KEY'),

    /*
    |--------------------------------------------------------------------------
    | Password Grant Client
    |--------------------------------------------------------------------------
    |
    | These values are used by first-party authentication endpoints that
    | proxy token issuance through Passport's /oauth/token endpoint.
    |
    */

    'password_client_id' => env('PASSPORT_PASSWORD_CLIENT_ID'),

    'password_client_secret' => env('PASSPORT_PASSWORD_CLIENT_SECRET'),

    /*
    |--------------------------------------------------------------------------
    | Token Lifetimes
    |--------------------------------------------------------------------------
    |
    | Lifetimes are applied in AppServiceProvider via Passport::tokensExpireIn
    | and Passport::refreshTokensExpireIn.
    |
    */

    'access_token_ttl_minutes' => (int) env('PASSPORT_ACCESS_TOKEN_TTL_MINUTES', 15),

    'refresh_token_ttl_days' => (int) env('PASSPORT_REFRESH_TOKEN_TTL_DAYS', 30),

    'personal_access_token_ttl_days' => (int) env('PASSPORT_PERSONAL_ACCESS_TOKEN_TTL_DAYS', 365),

    /*
    |--------------------------------------------------------------------------
    | Passport Database Connection
    |--------------------------------------------------------------------------
    |
    | By default, Passport's models will utilize your application's default
    | database connection. If you wish to use a different connection you
    | may specify the configured name of the database connection here.
    |
    */

    'connection' => env('PASSPORT_CONNECTION'),

];
