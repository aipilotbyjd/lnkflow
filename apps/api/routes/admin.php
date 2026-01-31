<?php

use App\Http\Controllers\Admin\AuthController;
use Illuminate\Support\Facades\Route;

Route::post('login', [AuthController::class, 'login'])->name('admin.login');

Route::middleware('auth:admin')->group(function () {
    Route::post('logout', [AuthController::class, 'logout'])->name('admin.logout');
    Route::get('me', [AuthController::class, 'me'])->name('admin.me');
});
