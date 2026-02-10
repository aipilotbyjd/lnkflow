<?php

namespace App\Providers;

use Illuminate\Support\ServiceProvider;
use Laravel\Passport\Passport;

class AppServiceProvider extends ServiceProvider
{
    /**
     * Register any application services.
     */
    public function register(): void
    {
        //
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        Passport::enablePasswordGrant();

        $accessTokenTtlMinutes = max((int) config('passport.access_token_ttl_minutes', 15), 1);
        $refreshTokenTtlDays = max((int) config('passport.refresh_token_ttl_days', 30), 1);
        $personalTokenTtlDays = max((int) config('passport.personal_access_token_ttl_days', 365), 1);

        Passport::tokensExpireIn(now()->addMinutes($accessTokenTtlMinutes));
        Passport::refreshTokensExpireIn(now()->addDays($refreshTokenTtlDays));
        Passport::personalAccessTokensExpireIn(now()->addDays($personalTokenTtlDays));
    }
}
