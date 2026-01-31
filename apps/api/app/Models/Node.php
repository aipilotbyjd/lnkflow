<?php

namespace App\Models;

use App\Enums\NodeKind;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class Node extends Model
{
    /** @use HasFactory<\Database\Factories\NodeFactory> */
    use HasFactory;

    protected $fillable = [
        'category_id',
        'type',
        'name',
        'description',
        'icon',
        'color',
        'node_kind',
        'config_schema',
        'output_schema',
        'credential_type',
        'is_active',
        'is_premium',
        'docs_url',
    ];

    /**
     * @return array<string, string>
     */
    protected function casts(): array
    {
        return [
            'node_kind' => NodeKind::class,
            'config_schema' => 'array',
            'output_schema' => 'array',
            'is_active' => 'boolean',
            'is_premium' => 'boolean',
        ];
    }

    /**
     * @return BelongsTo<NodeCategory, $this>
     */
    public function category(): BelongsTo
    {
        return $this->belongsTo(NodeCategory::class, 'category_id');
    }

    public function scopeActive($query)
    {
        return $query->where('is_active', true);
    }

    public function scopeFree($query)
    {
        return $query->where('is_premium', false);
    }

    public function scopeByKind($query, NodeKind $kind)
    {
        return $query->where('node_kind', $kind);
    }

    public function isTrigger(): bool
    {
        return $this->node_kind === NodeKind::Trigger;
    }

    public function requiresCredential(): bool
    {
        return $this->credential_type !== null;
    }
}
