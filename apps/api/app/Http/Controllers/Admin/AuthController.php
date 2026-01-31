<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Http\Requests\Admin\Auth\LoginRequest;
use App\Http\Resources\Admin\AdminResource;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;

class AuthController extends Controller
{
    public function login(LoginRequest $request): JsonResponse
    {
        if (! Auth::guard('admin')->attempt($request->validated(), $request->boolean('remember'))) {
            return response()->json([
                'message' => 'Invalid credentials.',
            ], 401);
        }

        $request->session()->regenerate();

        return response()->json([
            'message' => 'Login successful.',
            'admin' => new AdminResource(Auth::guard('admin')->user()),
        ]);
    }

    public function logout(Request $request): JsonResponse
    {
        Auth::guard('admin')->logout();

        $request->session()->invalidate();
        $request->session()->regenerateToken();

        return response()->json([
            'message' => 'Logged out successfully.',
        ]);
    }

    public function me(Request $request): JsonResponse
    {
        return response()->json([
            'admin' => new AdminResource($request->user('admin')),
        ]);
    }
}
