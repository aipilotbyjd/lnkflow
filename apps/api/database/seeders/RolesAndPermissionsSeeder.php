<?php

namespace Database\Seeders;

use Illuminate\Database\Seeder;
use Spatie\Permission\Models\Permission;
use Spatie\Permission\Models\Role;
use Spatie\Permission\PermissionRegistrar;

class RolesAndPermissionsSeeder extends Seeder
{
    public function run(): void
    {
        app()[PermissionRegistrar::class]->forgetCachedPermissions();

        $permissions = [
            // Workspace permissions
            'workspace.view',
            'workspace.update',
            'workspace.delete',
            'workspace.manage-billing',

            // Member permissions
            'member.view',
            'member.invite',
            'member.update',
            'member.remove',

            // Workflow permissions
            'workflow.view',
            'workflow.create',
            'workflow.update',
            'workflow.delete',
            'workflow.execute',

            // Credential permissions
            'credential.view',
            'credential.create',
            'credential.update',
            'credential.delete',

            // Execution permissions
            'execution.view',
            'execution.delete',
        ];

        foreach ($permissions as $permission) {
            Permission::create(['name' => $permission, 'guard_name' => 'api']);
        }

        // Owner - Full access
        Role::create(['name' => 'owner', 'guard_name' => 'api'])
            ->givePermissionTo(Permission::all());

        // Admin - All except billing and workspace delete
        Role::create(['name' => 'admin', 'guard_name' => 'api'])
            ->givePermissionTo([
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
            ]);

        // Member - Create and manage own workflows
        Role::create(['name' => 'member', 'guard_name' => 'api'])
            ->givePermissionTo([
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
            ]);

        // Viewer - Read-only access
        Role::create(['name' => 'viewer', 'guard_name' => 'api'])
            ->givePermissionTo([
                'workspace.view',
                'member.view',
                'workflow.view',
                'credential.view',
                'execution.view',
            ]);
    }
}
