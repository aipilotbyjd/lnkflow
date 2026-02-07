<?php

use App\Models\User;
use Illuminate\Foundation\Testing\RefreshDatabase;

uses(RefreshDatabase::class);

describe('Authentication', function () {
    describe('Registration', function () {
        it('can register a new user', function () {
            $response = $this->postJson('/api/v1/auth/register', [
                'first_name' => 'John',
                'last_name' => 'Doe',
                'email' => 'john@example.com',
                'password' => 'password123',
                'password_confirmation' => 'password123',
            ]);

            $response->assertStatus(201)
                ->assertJsonStructure([
                    'message',
                    'user' => ['id', 'first_name', 'last_name', 'email'],
                ]);

            $this->assertDatabaseHas('users', [
                'email' => 'john@example.com',
            ]);
        });

        it('fails with invalid email', function () {
            $response = $this->postJson('/api/v1/auth/register', [
                'first_name' => 'John',
                'last_name' => 'Doe',
                'email' => 'invalid-email',
                'password' => 'password123',
                'password_confirmation' => 'password123',
            ]);

            $response->assertStatus(422)
                ->assertJsonValidationErrors(['email']);
        });

        it('fails with duplicate email', function () {
            User::factory()->create(['email' => 'john@example.com']);

            $response = $this->postJson('/api/v1/auth/register', [
                'first_name' => 'John',
                'last_name' => 'Doe',
                'email' => 'john@example.com',
                'password' => 'password123',
                'password_confirmation' => 'password123',
            ]);

            $response->assertStatus(422)
                ->assertJsonValidationErrors(['email']);
        });

        it('fails with weak password', function () {
            $response = $this->postJson('/api/v1/auth/register', [
                'first_name' => 'John',
                'last_name' => 'Doe',
                'email' => 'john@example.com',
                'password' => '123',
                'password_confirmation' => '123',
            ]);

            $response->assertStatus(422)
                ->assertJsonValidationErrors(['password']);
        });
    });

    describe('Login', function () {
        it('can login with valid credentials', function () {
            $user = User::factory()->create([
                'password' => bcrypt('password123'),
            ]);

            $response = $this->postJson('/api/v1/auth/login', [
                'email' => $user->email,
                'password' => 'password123',
            ]);

            $response->assertStatus(200)
                ->assertJsonStructure([
                    'access_token',
                    'token_type',
                    'expires_in',
                    'user',
                ]);
        });

        it('fails with invalid credentials', function () {
            $user = User::factory()->create([
                'password' => bcrypt('password123'),
            ]);

            $response = $this->postJson('/api/v1/auth/login', [
                'email' => $user->email,
                'password' => 'wrongpassword',
            ]);

            $response->assertStatus(401);
        });

        it('fails with non-existent user', function () {
            $response = $this->postJson('/api/v1/auth/login', [
                'email' => 'nonexistent@example.com',
                'password' => 'password123',
            ]);

            $response->assertStatus(401);
        });
    });

    describe('Logout', function () {
        it('can logout authenticated user', function () {
            $user = User::factory()->create();

            $response = $this->actingAs($user, 'api')
                ->postJson('/api/v1/auth/logout');

            $response->assertStatus(200)
                ->assertJson(['message' => 'Successfully logged out']);
        });

        it('requires authentication to logout', function () {
            $response = $this->postJson('/api/v1/auth/logout');

            $response->assertStatus(401);
        });
    });

    describe('Password Reset', function () {
        it('can request password reset', function () {
            $user = User::factory()->create();

            $response = $this->postJson('/api/v1/auth/forgot-password', [
                'email' => $user->email,
            ]);

            $response->assertStatus(200)
                ->assertJsonStructure(['message']);
        });

        it('does not reveal non-existent emails', function () {
            $response = $this->postJson('/api/v1/auth/forgot-password', [
                'email' => 'nonexistent@example.com',
            ]);

            // Should still return 200 to prevent email enumeration
            $response->assertStatus(200);
        });
    });
});

describe('User Profile', function () {
    it('can get current user profile', function () {
        $user = User::factory()->create();

        $response = $this->actingAs($user, 'api')
            ->getJson('/api/v1/user');

        $response->assertStatus(200)
            ->assertJsonStructure([
                'user' => ['id', 'first_name', 'last_name', 'email'],
            ]);
    });

    it('can update user profile', function () {
        $user = User::factory()->create();

        $response = $this->actingAs($user, 'api')
            ->putJson('/api/v1/user', [
                'first_name' => 'Updated',
                'last_name' => 'Name',
            ]);

        $response->assertStatus(200);

        $this->assertDatabaseHas('users', [
            'id' => $user->id,
            'first_name' => 'Updated',
            'last_name' => 'Name',
        ]);
    });

    it('can change password', function () {
        $user = User::factory()->create([
            'password' => bcrypt('oldpassword'),
        ]);

        $response = $this->actingAs($user, 'api')
            ->putJson('/api/v1/user/password', [
                'current_password' => 'oldpassword',
                'password' => 'newpassword123',
                'password_confirmation' => 'newpassword123',
            ]);

        $response->assertStatus(200);
    });

    it('fails to change password with wrong current password', function () {
        $user = User::factory()->create([
            'password' => bcrypt('oldpassword'),
        ]);

        $response = $this->actingAs($user, 'api')
            ->putJson('/api/v1/user/password', [
                'current_password' => 'wrongpassword',
                'password' => 'newpassword123',
                'password_confirmation' => 'newpassword123',
            ]);

        $response->assertStatus(422);
    });

    it('requires authentication for profile access', function () {
        $response = $this->getJson('/api/v1/user');

        $response->assertStatus(401);
    });
});
