<?php

namespace Database\Seeders;

use Illuminate\Database\Seeder;

class RolesAndPermissionsSeeder extends Seeder
{
    public function run(): void
    {
        // Roles and permissions are managed by WorkspacePermissionService.
        // Each workspace member has a role (owner, admin, member, viewer)
        // stored in the workspace_members pivot table.
        // No seeding is required — permissions are defined in code.
    }
}
