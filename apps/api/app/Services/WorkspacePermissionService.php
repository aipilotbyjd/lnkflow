<?php

namespace App\Services;

use App\Models\User;
use App\Models\Workspace;

class WorkspacePermissionService
{
    /**
     * Permission mapping for each role
     *
     * @var array<string, array<string>>
     */
    private const ROLE_PERMISSIONS = [
        'owner' => [
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
            'credential.view',
            'credential.create',
            'credential.update',
            'credential.delete',
            'execution.view',
            'execution.delete',
        ],
        'admin' => [
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
            'credential.view',
            'credential.create',
            'credential.update',
            'credential.delete',
            'execution.view',
            'execution.delete',
        ],
        'member' => [
            'workspace.view',
            'member.view',
            'workflow.view',
            'workflow.create',
            'workflow.update',
            'workflow.delete',
            'workflow.execute',
            'credential.view',
            'credential.create',
            'credential.update',
            'credential.delete',
            'execution.view',
        ],
        'viewer' => [
            'workspace.view',
            'member.view',
            'workflow.view',
            'credential.view',
            'execution.view',
        ],
    ];

    public function getUserRoleInWorkspace(User $user, Workspace $workspace): ?string
    {
        $member = $workspace->members()->where('user_id', $user->id)->first();

        return $member?->pivot?->role;
    }

    public function isMember(User $user, Workspace $workspace): bool
    {
        return $workspace->members()->where('user_id', $user->id)->exists();
    }

    public function isOwner(User $user, Workspace $workspace): bool
    {
        return $workspace->owner_id === $user->id;
    }

    public function hasPermission(User $user, Workspace $workspace, string $permission): bool
    {
        $role = $this->getUserRoleInWorkspace($user, $workspace);

        if (! $role) {
            return false;
        }

        $permissions = self::ROLE_PERMISSIONS[$role] ?? [];

        return in_array($permission, $permissions, true);
    }

    public function hasAnyPermission(User $user, Workspace $workspace, array $permissions): bool
    {
        foreach ($permissions as $permission) {
            if ($this->hasPermission($user, $workspace, $permission)) {
                return true;
            }
        }

        return false;
    }

    public function hasAllPermissions(User $user, Workspace $workspace, array $permissions): bool
    {
        foreach ($permissions as $permission) {
            if (! $this->hasPermission($user, $workspace, $permission)) {
                return false;
            }
        }

        return true;
    }

    public function authorize(User $user, Workspace $workspace, string $permission): void
    {
        if (! $this->hasPermission($user, $workspace, $permission)) {
            abort(403, 'You do not have permission to perform this action.');
        }
    }

    public function authorizeMembership(User $user, Workspace $workspace): void
    {
        if (! $this->isMember($user, $workspace)) {
            abort(403, 'You are not a member of this workspace.');
        }
    }

    /**
     * @return array<string>
     */
    public function getPermissionsForRole(string $role): array
    {
        return self::ROLE_PERMISSIONS[$role] ?? [];
    }
}
