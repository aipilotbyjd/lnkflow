<?php

use Illuminate\Support\Facades\Schedule;

/*
|--------------------------------------------------------------------------
| Console Routes
|--------------------------------------------------------------------------
|
| This file is where you may define all of your Closure based console
| commands. Each Closure is bound to a command instance allowing a
| simple approach to interacting with each command's IO methods.
|
*/

// Dispatch scheduled workflows every minute
Schedule::command('workflows:dispatch-scheduled')
    ->everyMinute()
    ->withoutOverlapping()
    ->runInBackground()
    ->appendOutputTo(storage_path('logs/scheduler.log'));

// Clean up old executions (keep 30 days)
Schedule::command('executions:cleanup --days=30')
    ->dailyAt('02:00')
    ->withoutOverlapping();

// Send workflow failure notifications digest
Schedule::command('notifications:send-digest')
    ->hourly()
    ->withoutOverlapping();

// Roll up connector reliability metrics daily
Schedule::command('connectors:rollup-metrics')
    ->dailyAt('01:30')
    ->withoutOverlapping();
