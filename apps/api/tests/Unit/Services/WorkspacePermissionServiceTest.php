<?php

use App\Models\User;
use App\Models\Workspace;
use App\Services\WorkspacePermissionService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Symfony\Component\HttpKernel\Exception\HttpException;

uses(RefreshDatabase::class);

beforeEach(function () {
    $this->service = new WorkspacePermissionService;

    $this->owner = User::factory()->create();
    $this->workspace = Workspace::factory()->create(['owner_id' => $this->owner->id]);
    $this->workspace->members()->attach($this->owner->id, ['role' => 'owner', 'joined_at' => now()]);

    $this->admin = User::factory()->create();
    $this->workspace->members()->attach($this->admin->id, ['role' => 'admin', 'joined_at' => now()]);

    $this->member = User::factory()->create();
    $this->workspace->members()->attach($this->member->id, ['role' => 'member', 'joined_at' => now()]);

    $this->viewer = User::factory()->create();
    $this->workspace->members()->attach($this->viewer->id, ['role' => 'viewer', 'joined_at' => now()]);

    $this->nonMember = User::factory()->create();
});

// ──────────────────────────────────────────────────────
// 1. Owner has ALL permissions
// ──────────────────────────────────────────────────────
describe('Owner permissions', function () {
    it('has all permissions including workspace.delete and workspace.manage-billing', function () {
        $allOwnerPermissions = [
            'workspace.view',
            'workspace.update',
            'workspace.delete',
            'workspace.manage-billing',
            'member.view',
            'member.invite',
            'member.update',
            'member.remove',
            'workflow.view',
            'workflow.create',
            'workflow.update',
            'workflow.delete',
            'workflow.execute',
            'workflow.activate',
            'workflow.export',
            'workflow.import',
            'credential.view',
            'credential.create',
            'credential.update',
            'credential.delete',
            'execution.view',
            'execution.delete',
            'webhook.view',
            'webhook.create',
            'webhook.update',
            'webhook.delete',
            'variable.view',
            'variable.create',
            'variable.update',
            'variable.delete',
            'tag.view',
            'tag.create',
            'tag.update',
            'tag.delete',
            'environment.view',
            'environment.create',
            'environment.deploy',
        ];

        foreach ($allOwnerPermissions as $permission) {
            expect($this->service->hasPermission($this->owner, $this->workspace, $permission))
                ->toBeTrue("Owner should have permission: {$permission}");
        }
    });

    it('returns owner role for the workspace owner', function () {
        expect($this->service->getUserRoleInWorkspace($this->owner, $this->workspace))->toBe('owner');
    });

    it('is identified as owner via isOwner()', function () {
        expect($this->service->isOwner($this->owner, $this->workspace))->toBeTrue();
    });

    it('is identified as member via isMember()', function () {
        expect($this->service->isMember($this->owner, $this->workspace))->toBeTrue();
    });
});

// ──────────────────────────────────────────────────────
// 2. Admin has all permissions EXCEPT workspace.delete and workspace.manage-billing
// ──────────────────────────────────────────────────────
describe('Admin permissions', function () {
    it('has most permissions but not workspace.delete or workspace.manage-billing', function () {
        $adminAllowed = [
            'workspace.view',
            'workspace.update',
            'member.view',
            'member.invite',
            'member.update',
            'member.remove',
            'workflow.view',
            'workflow.create',
            'workflow.update',
            'workflow.delete',
            'workflow.execute',
            'workflow.activate',
            'workflow.export',
            'workflow.import',
            'credential.view',
            'credential.create',
            'credential.update',
            'credential.delete',
            'execution.view',
            'execution.delete',
            'webhook.view',
            'webhook.create',
            'webhook.update',
            'webhook.delete',
            'variable.view',
            'variable.create',
            'variable.update',
            'variable.delete',
            'tag.view',
            'tag.create',
            'tag.update',
            'tag.delete',
            'environment.view',
            'environment.create',
            'environment.deploy',
        ];

        foreach ($adminAllowed as $permission) {
            expect($this->service->hasPermission($this->admin, $this->workspace, $permission))
                ->toBeTrue("Admin should have permission: {$permission}");
        }
    });

    it('does NOT have workspace.delete', function () {
        expect($this->service->hasPermission($this->admin, $this->workspace, 'workspace.delete'))->toBeFalse();
    });

    it('does NOT have workspace.manage-billing', function () {
        expect($this->service->hasPermission($this->admin, $this->workspace, 'workspace.manage-billing'))->toBeFalse();
    });

    it('is not identified as owner', function () {
        expect($this->service->isOwner($this->admin, $this->workspace))->toBeFalse();
    });
});

// ──────────────────────────────────────────────────────
// 3. Member has view + create + update but NOT delete for most resources
// ──────────────────────────────────────────────────────
describe('Member permissions', function () {
    it('has view, create, and update for workflows, credentials, webhooks, variables, tags', function () {
        $memberAllowed = [
            'workspace.view',
            'member.view',
            'workflow.view',
            'workflow.create',
            'workflow.update',
            'workflow.execute',
            'workflow.export',
            'credential.view',
            'credential.create',
            'credential.update',
            'execution.view',
            'webhook.view',
            'webhook.create',
            'webhook.update',
            'variable.view',
            'variable.create',
            'variable.update',
            'tag.view',
            'tag.create',
            'tag.update',
            'environment.view',
        ];

        foreach ($memberAllowed as $permission) {
            expect($this->service->hasPermission($this->member, $this->workspace, $permission))
                ->toBeTrue("Member should have permission: {$permission}");
        }
    });

    it('does NOT have delete permissions for workflows, credentials, webhooks, variables, tags', function () {
        $memberDenied = [
            'workflow.delete',
            'credential.delete',
            'execution.delete',
            'webhook.delete',
            'variable.delete',
            'tag.delete',
        ];

        foreach ($memberDenied as $permission) {
            expect($this->service->hasPermission($this->member, $this->workspace, $permission))
                ->toBeFalse("Member should NOT have permission: {$permission}");
        }
    });

    it('does NOT have workflow.activate', function () {
        expect($this->service->hasPermission($this->member, $this->workspace, 'workflow.activate'))->toBeFalse();
    });

    it('does NOT have environment.deploy', function () {
        expect($this->service->hasPermission($this->member, $this->workspace, 'environment.deploy'))->toBeFalse();
    });

    it('does NOT have workflow.import', function () {
        expect($this->service->hasPermission($this->member, $this->workspace, 'workflow.import'))->toBeFalse();
    });

    it('does NOT have member management permissions', function () {
        expect($this->service->hasPermission($this->member, $this->workspace, 'member.invite'))->toBeFalse();
        expect($this->service->hasPermission($this->member, $this->workspace, 'member.update'))->toBeFalse();
        expect($this->service->hasPermission($this->member, $this->workspace, 'member.remove'))->toBeFalse();
    });
});

// ──────────────────────────────────────────────────────
// 4. Viewer has only *.view permissions
// ──────────────────────────────────────────────────────
describe('Viewer permissions', function () {
    it('has all view permissions', function () {
        $viewPermissions = [
            'workspace.view',
            'member.view',
            'workflow.view',
            'credential.view',
            'execution.view',
            'webhook.view',
            'variable.view',
            'tag.view',
            'environment.view',
        ];

        foreach ($viewPermissions as $permission) {
            expect($this->service->hasPermission($this->viewer, $this->workspace, $permission))
                ->toBeTrue("Viewer should have permission: {$permission}");
        }
    });

    it('does NOT have any non-view permissions', function () {
        $nonViewPermissions = [
            'workspace.update',
            'workspace.delete',
            'workspace.manage-billing',
            'member.invite',
            'member.update',
            'member.remove',
            'workflow.create',
            'workflow.update',
            'workflow.delete',
            'workflow.execute',
            'workflow.activate',
            'workflow.export',
            'workflow.import',
            'credential.create',
            'credential.update',
            'credential.delete',
            'execution.delete',
            'webhook.create',
            'webhook.update',
            'webhook.delete',
            'variable.create',
            'variable.update',
            'variable.delete',
            'tag.create',
            'tag.update',
            'tag.delete',
            'environment.create',
            'environment.deploy',
        ];

        foreach ($nonViewPermissions as $permission) {
            expect($this->service->hasPermission($this->viewer, $this->workspace, $permission))
                ->toBeFalse("Viewer should NOT have permission: {$permission}");
        }
    });
});

// ──────────────────────────────────────────────────────
// 5. Non-member has NO permissions
// ──────────────────────────────────────────────────────
describe('Non-member permissions', function () {
    it('has no permissions at all', function () {
        expect($this->service->hasPermission($this->nonMember, $this->workspace, 'workspace.view'))->toBeFalse();
        expect($this->service->hasPermission($this->nonMember, $this->workspace, 'workflow.view'))->toBeFalse();
        expect($this->service->hasPermission($this->nonMember, $this->workspace, 'workflow.create'))->toBeFalse();
    });

    it('is not a member', function () {
        expect($this->service->isMember($this->nonMember, $this->workspace))->toBeFalse();
    });

    it('is not an owner', function () {
        expect($this->service->isOwner($this->nonMember, $this->workspace))->toBeFalse();
    });

    it('getUserRoleInWorkspace returns null', function () {
        expect($this->service->getUserRoleInWorkspace($this->nonMember, $this->workspace))->toBeNull();
    });
});

// ──────────────────────────────────────────────────────
// 6. Owner fallback – owner gets owner permissions even without pivot entry
// ──────────────────────────────────────────────────────
describe('Owner fallback without pivot', function () {
    it('grants owner permissions via hasPermission even when not in pivot table', function () {
        $ownerNoPivot = User::factory()->create();
        $workspace = Workspace::factory()->create(['owner_id' => $ownerNoPivot->id]);
        // Intentionally NOT attaching the owner to the members pivot

        expect($this->service->hasPermission($ownerNoPivot, $workspace, 'workspace.delete'))->toBeTrue();
        expect($this->service->hasPermission($ownerNoPivot, $workspace, 'workspace.manage-billing'))->toBeTrue();
        expect($this->service->hasPermission($ownerNoPivot, $workspace, 'workflow.view'))->toBeTrue();
    });

    it('grants owner permissions via hasAnyPermission even when not in pivot table', function () {
        $ownerNoPivot = User::factory()->create();
        $workspace = Workspace::factory()->create(['owner_id' => $ownerNoPivot->id]);

        expect($this->service->hasAnyPermission($ownerNoPivot, $workspace, ['workspace.delete', 'nonexistent.perm']))
            ->toBeTrue();
    });

    it('grants owner permissions via hasAllPermissions even when not in pivot table', function () {
        $ownerNoPivot = User::factory()->create();
        $workspace = Workspace::factory()->create(['owner_id' => $ownerNoPivot->id]);

        expect($this->service->hasAllPermissions($ownerNoPivot, $workspace, ['workspace.delete', 'workspace.manage-billing']))
            ->toBeTrue();
    });

    it('isOwner returns true but isMember returns false when not in pivot', function () {
        $ownerNoPivot = User::factory()->create();
        $workspace = Workspace::factory()->create(['owner_id' => $ownerNoPivot->id]);

        expect($this->service->isOwner($ownerNoPivot, $workspace))->toBeTrue();
        expect($this->service->isMember($ownerNoPivot, $workspace))->toBeFalse();
    });
});

// ──────────────────────────────────────────────────────
// 7. Role caching
// ──────────────────────────────────────────────────────
describe('Role caching', function () {
    it('caches the role after first lookup', function () {
        // First call queries the database
        $role1 = $this->service->getUserRoleInWorkspace($this->admin, $this->workspace);
        expect($role1)->toBe('admin');

        // Remove the member from the pivot (simulating DB change)
        $this->workspace->members()->detach($this->admin->id);

        // Second call should still return cached result
        $role2 = $this->service->getUserRoleInWorkspace($this->admin, $this->workspace);
        expect($role2)->toBe('admin');
    });

    it('clearCache resets the cached roles', function () {
        // Populate cache
        $this->service->getUserRoleInWorkspace($this->admin, $this->workspace);

        // Remove from pivot
        $this->workspace->members()->detach($this->admin->id);

        // Clear and re-query
        $this->service->clearCache();
        $role = $this->service->getUserRoleInWorkspace($this->admin, $this->workspace);
        expect($role)->toBeNull();
    });

    it('caches null (false internally) for non-members', function () {
        // First call – not a member
        $role1 = $this->service->getUserRoleInWorkspace($this->nonMember, $this->workspace);
        expect($role1)->toBeNull();

        // Attach user as a member now
        $this->workspace->members()->attach($this->nonMember->id, ['role' => 'viewer', 'joined_at' => now()]);

        // Still returns null because of cache
        $role2 = $this->service->getUserRoleInWorkspace($this->nonMember, $this->workspace);
        expect($role2)->toBeNull();

        // After clearing, returns the new role
        $this->service->clearCache();
        $role3 = $this->service->getUserRoleInWorkspace($this->nonMember, $this->workspace);
        expect($role3)->toBe('viewer');
    });

    it('uses user_id:workspace_id as the cache key format', function () {
        // Create a second workspace to ensure different cache keys
        $workspace2 = Workspace::factory()->create(['owner_id' => $this->admin->id]);
        $workspace2->members()->attach($this->admin->id, ['role' => 'owner', 'joined_at' => now()]);

        // Query both workspaces – they should not interfere
        $role1 = $this->service->getUserRoleInWorkspace($this->admin, $this->workspace);
        $role2 = $this->service->getUserRoleInWorkspace($this->admin, $workspace2);

        expect($role1)->toBe('admin');
        expect($role2)->toBe('owner');
    });
});

// ──────────────────────────────────────────────────────
// 8. hasAnyPermission
// ──────────────────────────────────────────────────────
describe('hasAnyPermission', function () {
    it('returns true if at least one permission matches', function () {
        expect($this->service->hasAnyPermission($this->viewer, $this->workspace, ['workspace.delete', 'workspace.view']))
            ->toBeTrue();
    });

    it('returns false if no permissions match', function () {
        expect($this->service->hasAnyPermission($this->viewer, $this->workspace, ['workspace.delete', 'workspace.manage-billing']))
            ->toBeFalse();
    });

    it('returns false for non-member', function () {
        expect($this->service->hasAnyPermission($this->nonMember, $this->workspace, ['workspace.view']))
            ->toBeFalse();
    });

    it('returns true for owner via resolveEffectiveRole even without pivot', function () {
        $ownerNoPivot = User::factory()->create();
        $workspace = Workspace::factory()->create(['owner_id' => $ownerNoPivot->id]);

        expect($this->service->hasAnyPermission($ownerNoPivot, $workspace, ['workspace.delete']))
            ->toBeTrue();
    });
});

// ──────────────────────────────────────────────────────
// 9. hasAllPermissions
// ──────────────────────────────────────────────────────
describe('hasAllPermissions', function () {
    it('returns true when user has all requested permissions', function () {
        expect($this->service->hasAllPermissions($this->owner, $this->workspace, ['workspace.view', 'workspace.delete', 'workspace.manage-billing']))
            ->toBeTrue();
    });

    it('returns false if any one permission is missing', function () {
        expect($this->service->hasAllPermissions($this->admin, $this->workspace, ['workspace.view', 'workspace.delete']))
            ->toBeFalse();
    });

    it('returns false for non-member', function () {
        expect($this->service->hasAllPermissions($this->nonMember, $this->workspace, ['workspace.view']))
            ->toBeFalse();
    });

    it('returns true for empty permissions array', function () {
        expect($this->service->hasAllPermissions($this->viewer, $this->workspace, []))
            ->toBeTrue();
    });

    it('returns false when member lacks one of many permissions', function () {
        expect($this->service->hasAllPermissions($this->member, $this->workspace, ['workflow.view', 'workflow.delete']))
            ->toBeFalse();
    });
});

// ──────────────────────────────────────────────────────
// 10. getPermissionsForRole
// ──────────────────────────────────────────────────────
describe('getPermissionsForRole', function () {
    it('returns permissions array for owner role', function () {
        $permissions = $this->service->getPermissionsForRole('owner');
        expect($permissions)->toBeArray();
        expect($permissions)->toContain('workspace.delete');
        expect($permissions)->toContain('workspace.manage-billing');
    });

    it('returns permissions array for admin role', function () {
        $permissions = $this->service->getPermissionsForRole('admin');
        expect($permissions)->toBeArray();
        expect($permissions)->not->toContain('workspace.delete');
        expect($permissions)->not->toContain('workspace.manage-billing');
        expect($permissions)->toContain('workflow.delete');
    });

    it('returns permissions array for member role', function () {
        $permissions = $this->service->getPermissionsForRole('member');
        expect($permissions)->toBeArray();
        expect($permissions)->toContain('workflow.view');
        expect($permissions)->not->toContain('workflow.delete');
    });

    it('returns permissions array for viewer role', function () {
        $permissions = $this->service->getPermissionsForRole('viewer');
        expect($permissions)->toBeArray();
        expect($permissions)->toContain('workspace.view');
        expect($permissions)->not->toContain('workspace.update');
    });

    it('returns empty array for unknown role', function () {
        expect($this->service->getPermissionsForRole('superadmin'))->toBe([]);
        expect($this->service->getPermissionsForRole(''))->toBe([]);
        expect($this->service->getPermissionsForRole('nonexistent'))->toBe([]);
    });
});

// ──────────────────────────────────────────────────────
// 11. getValidRoles
// ──────────────────────────────────────────────────────
describe('getValidRoles', function () {
    it('returns the four valid roles', function () {
        $roles = $this->service->getValidRoles();
        expect($roles)->toBe(['owner', 'admin', 'member', 'viewer']);
    });

    it('returns exactly 4 roles', function () {
        expect($this->service->getValidRoles())->toHaveCount(4);
    });
});

// ──────────────────────────────────────────────────────
// 12. authorize() throws HttpException 403
// ──────────────────────────────────────────────────────
describe('authorize', function () {
    it('does not throw when user has the permission', function () {
        // Should not throw
        $this->service->authorize($this->owner, $this->workspace, 'workspace.delete');
        expect(true)->toBeTrue(); // If we reach here, no exception was thrown
    });

    it('throws HttpException with 403 when user lacks the permission', function () {
        $this->expectException(HttpException::class);
        $this->service->authorize($this->viewer, $this->workspace, 'workspace.delete');
    });

    it('throws 403 for non-member', function () {
        try {
            $this->service->authorize($this->nonMember, $this->workspace, 'workspace.view');
            $this->fail('Expected HttpException was not thrown');
        } catch (HttpException $e) {
            expect($e->getStatusCode())->toBe(403);
            expect($e->getMessage())->toBe('You do not have permission to perform this action.');
        }
    });

    it('throws 403 for admin trying owner-only permission', function () {
        try {
            $this->service->authorize($this->admin, $this->workspace, 'workspace.delete');
            $this->fail('Expected HttpException was not thrown');
        } catch (HttpException $e) {
            expect($e->getStatusCode())->toBe(403);
        }
    });
});

// ──────────────────────────────────────────────────────
// 13. authorizeMembership() throws HttpException 403
// ──────────────────────────────────────────────────────
describe('authorizeMembership', function () {
    it('does not throw when user is a member', function () {
        $this->service->authorizeMembership($this->owner, $this->workspace);
        $this->service->authorizeMembership($this->admin, $this->workspace);
        $this->service->authorizeMembership($this->member, $this->workspace);
        $this->service->authorizeMembership($this->viewer, $this->workspace);
        expect(true)->toBeTrue();
    });

    it('throws HttpException with 403 for non-member', function () {
        $this->expectException(HttpException::class);
        $this->service->authorizeMembership($this->nonMember, $this->workspace);
    });

    it('throws 403 with correct message for non-member', function () {
        try {
            $this->service->authorizeMembership($this->nonMember, $this->workspace);
            $this->fail('Expected HttpException was not thrown');
        } catch (HttpException $e) {
            expect($e->getStatusCode())->toBe(403);
            expect($e->getMessage())->toBe('You are not a member of this workspace.');
        }
    });

    it('throws 403 for owner without pivot entry', function () {
        $ownerNoPivot = User::factory()->create();
        $workspace = Workspace::factory()->create(['owner_id' => $ownerNoPivot->id]);

        // Owner is not in the members pivot, so isMember() returns false
        $this->expectException(HttpException::class);
        $this->service->authorizeMembership($ownerNoPivot, $workspace);
    });
});
