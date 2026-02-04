// LinkFlow Workflow Editor
// This editor creates WorkflowDefinition compatible with the Go engine

const state = {
    nodes: [],
    edges: [],
    selectedNode: null,
    draggingNode: null,
    connecting: null,
    offset: { x: 0, y: 0 },
    nodeCounter: 0,
    // Auth state
    auth: {
        token: null,
        user: null,
        workspaces: [],
        workflows: [],
        currentWorkspace: null,
        currentWorkflow: null
    }
};

const canvas = document.getElementById('canvas');
const edgesSvg = document.getElementById('edges-svg');

// Node type configurations
const nodeTypes = {
    trigger_manual: { icon: '▶️', label: 'Manual Trigger', category: 'trigger' },
    trigger_webhook: { icon: '🔗', label: 'Webhook', category: 'trigger' },
    trigger_schedule: { icon: '⏰', label: 'Schedule', category: 'trigger' },
    trigger_event: { icon: '📡', label: 'Event', category: 'trigger' },
    action_http: { icon: '🌐', label: 'HTTP Request', category: 'action' },
    action_script: { icon: '📝', label: 'Script', category: 'action' },
    action_transform: { icon: '🔄', label: 'Transform', category: 'action' },
    action_delay: { icon: '⏳', label: 'Delay', category: 'action' },
    logic_condition: { icon: '❓', label: 'Condition', category: 'logic' },
    logic_switch: { icon: '🔀', label: 'Switch', category: 'logic' },
    logic_loop: { icon: '🔁', label: 'Loop', category: 'logic' },
    logic_parallel: { icon: '⚡', label: 'Parallel', category: 'logic' },
    output_response: { icon: '📤', label: 'Response', category: 'output' },
    output_log: { icon: '📋', label: 'Log', category: 'output' }
};

// Initialize drag from palette
document.querySelectorAll('.palette-node').forEach(node => {
    node.addEventListener('dragstart', (e) => {
        e.dataTransfer.setData('nodeType', e.target.dataset.type);
    });
});

canvas.addEventListener('dragover', (e) => e.preventDefault());

canvas.addEventListener('drop', (e) => {
    e.preventDefault();
    const nodeType = e.dataTransfer.getData('nodeType');
    if (nodeType) {
        const rect = canvas.getBoundingClientRect();
        createNode(nodeType, e.clientX - rect.left - 90, e.clientY - rect.top - 30);
    }
});

function createNode(type, x, y) {
    const id = `node_${++state.nodeCounter}`;
    const config = nodeTypes[type];

    const node = {
        id,
        type,
        position: { x, y },
        data: {
            label: config.label,
            config: {}
        }
    };

    state.nodes.push(node);
    renderNode(node);
    updateOutput();
    setStatus(`Created ${config.label} node`);
}

function renderNode(node) {
    const config = nodeTypes[node.type];
    const el = document.createElement('div');
    el.className = `workflow-node ${config.category}`;
    el.id = node.id;
    el.style.left = `${node.position.x}px`;
    el.style.top = `${node.position.y}px`;

    const isTrigger = node.type.startsWith('trigger_');
    const isOutput = node.type.startsWith('output_');

    el.innerHTML = `
        ${!isTrigger ? '<div class="handle input"></div>' : ''}
        <div class="node-header">
            <span class="icon">${config.icon}</span>
            <span class="title">${node.data.label}</span>
        </div>
        <div class="node-body">
            <small style="color:#64748b">${node.id}</small>
        </div>
        ${!isOutput ? '<div class="handle output"></div>' : ''}
    `;

    // Node dragging
    el.addEventListener('mousedown', (e) => {
        if (e.target.classList.contains('handle')) return;
        selectNode(node.id);
        state.draggingNode = node.id;
        const rect = el.getBoundingClientRect();
        state.offset = { x: e.clientX - rect.left, y: e.clientY - rect.top };
    });

    // Handle connection start
    const outputHandle = el.querySelector('.handle.output');
    if (outputHandle) {
        outputHandle.addEventListener('mousedown', (e) => {
            e.stopPropagation();
            state.connecting = { source: node.id, x: e.clientX, y: e.clientY };
        });
    }

    // Handle connection end
    const inputHandle = el.querySelector('.handle.input');
    if (inputHandle) {
        inputHandle.addEventListener('mouseup', (e) => {
            e.stopPropagation();
            if (state.connecting && state.connecting.source !== node.id) {
                createEdge(state.connecting.source, node.id);
            }
            state.connecting = null;
            removeTempEdge();
        });
    }

    canvas.appendChild(el);
}

function selectNode(id) {
    document.querySelectorAll('.workflow-node').forEach(n => n.classList.remove('selected'));
    if (id) {
        document.getElementById(id)?.classList.add('selected');
        state.selectedNode = id;
    }
}

function createEdge(source, target) {
    // Check for duplicate
    const exists = state.edges.some(e => e.source === source && e.target === target);
    if (exists) return;

    // Check for self-loop
    if (source === target) return;

    const edge = {
        id: `edge_${source}_${target}`,
        source,
        target,
        sourceHandle: 'output',
        targetHandle: 'input'
    };

    state.edges.push(edge);
    renderEdge(edge);
    updateOutput();
    setStatus(`Connected ${source} → ${target}`);
}

function renderEdge(edge) {
    const sourceEl = document.getElementById(edge.source);
    const targetEl = document.getElementById(edge.target);
    if (!sourceEl || !targetEl) return;

    const sourceHandle = sourceEl.querySelector('.handle.output');
    const targetHandle = targetEl.querySelector('.handle.input');
    if (!sourceHandle || !targetHandle) return;

    const sourceRect = sourceHandle.getBoundingClientRect();
    const targetRect = targetHandle.getBoundingClientRect();
    const canvasRect = canvas.getBoundingClientRect();

    const x1 = sourceRect.left + sourceRect.width/2 - canvasRect.left;
    const y1 = sourceRect.top + sourceRect.height/2 - canvasRect.top;
    const x2 = targetRect.left + targetRect.width/2 - canvasRect.left;
    const y2 = targetRect.top + targetRect.height/2 - canvasRect.top;

    const path = createBezierPath(x1, y1, x2, y2);

    let pathEl = edgesSvg.querySelector(`#${edge.id}`);
    if (!pathEl) {
        pathEl = document.createElementNS('http://www.w3.org/2000/svg', 'path');
        pathEl.id = edge.id;
        pathEl.classList.add('edge');
        edgesSvg.appendChild(pathEl);
    }
    pathEl.setAttribute('d', path);
}

function createBezierPath(x1, y1, x2, y2) {
    const dx = Math.abs(x2 - x1) * 0.5;
    return `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
}

function renderAllEdges() {
    state.edges.forEach(renderEdge);
}

function removeTempEdge() {
    const temp = edgesSvg.querySelector('.edge-temp');
    if (temp) temp.remove();
}

// Mouse move handler
document.addEventListener('mousemove', (e) => {
    // Node dragging
    if (state.draggingNode) {
        const canvasRect = canvas.getBoundingClientRect();
        const node = state.nodes.find(n => n.id === state.draggingNode);
        if (node) {
            node.position.x = e.clientX - canvasRect.left - state.offset.x;
            node.position.y = e.clientY - canvasRect.top - state.offset.y;
            const el = document.getElementById(node.id);
            el.style.left = `${node.position.x}px`;
            el.style.top = `${node.position.y}px`;
            renderAllEdges();
        }
    }

    // Connection drawing
    if (state.connecting) {
        const sourceEl = document.getElementById(state.connecting.source);
        const sourceHandle = sourceEl.querySelector('.handle.output');
        const sourceRect = sourceHandle.getBoundingClientRect();
        const canvasRect = canvas.getBoundingClientRect();

        const x1 = sourceRect.left + sourceRect.width/2 - canvasRect.left;
        const y1 = sourceRect.top + sourceRect.height/2 - canvasRect.top;
        const x2 = e.clientX - canvasRect.left;
        const y2 = e.clientY - canvasRect.top;

        let tempEdge = edgesSvg.querySelector('.edge-temp');
        if (!tempEdge) {
            tempEdge = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            tempEdge.classList.add('edge-temp');
            edgesSvg.appendChild(tempEdge);
        }
        tempEdge.setAttribute('d', createBezierPath(x1, y1, x2, y2));
    }
});

document.addEventListener('mouseup', () => {
    if (state.draggingNode) {
        updateOutput();
    }
    state.draggingNode = null;
    state.connecting = null;
    removeTempEdge();
});

// Keyboard handlers
document.addEventListener('keydown', (e) => {
    if (e.key === 'Delete' || e.key === 'Backspace') {
        if (state.selectedNode && document.activeElement.tagName !== 'INPUT') {
            deleteNode(state.selectedNode);
        }
    }
});

function deleteNode(id) {
    // Remove node
    state.nodes = state.nodes.filter(n => n.id !== id);
    document.getElementById(id)?.remove();

    // Remove connected edges
    state.edges = state.edges.filter(e => {
        if (e.source === id || e.target === id) {
            edgesSvg.querySelector(`#${e.id}`)?.remove();
            return false;
        }
        return true;
    });

    state.selectedNode = null;
    updateOutput();
    setStatus(`Deleted node ${id}`);
}

// Export workflow definition (compatible with Go engine)
function getWorkflowDefinition() {
    return {
        id: 'workflow_' + Date.now(),
        name: 'Untitled Workflow',
        nodes: state.nodes.map(n => ({
            id: n.id,
            type: n.type,
            position: { x: n.position.x, y: n.position.y },
            data: {
                label: n.data.label,
                config: n.data.config || {}
            }
        })),
        edges: state.edges.map(e => ({
            id: e.id,
            source: e.source,
            target: e.target,
            sourceHandle: e.sourceHandle,
            targetHandle: e.targetHandle
        }))
    };
}

function updateOutput() {
    const workflow = getWorkflowDefinition();
    document.getElementById('json-output').textContent = JSON.stringify(workflow, null, 2);
    updateDagInfo(workflow);
}

function updateDagInfo(workflow) {
    const dagOutput = document.getElementById('dag-output');

    // Build simple DAG analysis
    const edges = {};
    const reverseEdges = {};
    workflow.nodes.forEach(n => {
        edges[n.id] = [];
        reverseEdges[n.id] = [];
    });
    workflow.edges.forEach(e => {
        edges[e.source].push(e.target);
        reverseEdges[e.target].push(e.source);
    });

    // Find entry/exit nodes
    const entryNodes = workflow.nodes.filter(n => reverseEdges[n.id].length === 0).map(n => n.id);
    const exitNodes = workflow.nodes.filter(n => edges[n.id].length === 0).map(n => n.id);

    // Compute levels
    const levels = {};
    const visited = new Set();
    const queue = [...entryNodes];
    entryNodes.forEach(id => levels[id] = 0);

    while (queue.length > 0) {
        const id = queue.shift();
        if (visited.has(id)) continue;
        visited.add(id);

        edges[id].forEach(next => {
            const newLevel = (levels[id] || 0) + 1;
            if (!levels[next] || levels[next] < newLevel) {
                levels[next] = newLevel;
            }
            queue.push(next);
        });
    }

    const maxLevel = Math.max(0, ...Object.values(levels));

    dagOutput.innerHTML = `
        <pre style="background:#0d1b2a;padding:12px;border-radius:8px;margin-bottom:12px;">
DAG Analysis
============
Total Nodes: ${workflow.nodes.length}
Total Edges: ${workflow.edges.length}

Entry Nodes: ${entryNodes.join(', ') || 'none'}
Exit Nodes:  ${exitNodes.join(', ') || 'none'}

Max Parallel Levels: ${maxLevel + 1}

Levels:
${Object.entries(levels).map(([id, lvl]) => `  Level ${lvl}: ${id}`).join('\n')}
        </pre>
    `;
}

function validateWorkflow() {
    const workflow = getWorkflowDefinition();
    const errors = [];

    // Check for empty workflow
    if (workflow.nodes.length === 0) {
        errors.push({ message: 'Workflow has no nodes' });
    }

    // Check for trigger
    const triggers = workflow.nodes.filter(n => n.type.startsWith('trigger_'));
    if (triggers.length === 0) {
        errors.push({ message: 'Workflow must have at least one trigger node' });
    }

    // Check for isolated nodes
    const connectedNodes = new Set();
    workflow.edges.forEach(e => {
        connectedNodes.add(e.source);
        connectedNodes.add(e.target);
    });
    workflow.nodes.forEach(n => {
        if (!connectedNodes.has(n.id) && workflow.nodes.length > 1) {
            errors.push({ nodeId: n.id, message: `Node ${n.id} is isolated (no connections)` });
        }
    });

    // Check for cycles (simple DFS)
    const hasCycle = detectCycle(workflow);
    if (hasCycle) {
        errors.push({ message: 'Cycle detected in workflow graph' });
    }

    showTab('validation');
    const validationOutput = document.getElementById('validation-output');

    if (errors.length === 0) {
        validationOutput.innerHTML = '<li class="validation-item success">✓ Workflow is valid!</li>';
        setStatus('Workflow validated successfully');
    } else {
        validationOutput.innerHTML = errors.map(e =>
            `<li class="validation-item">✗ ${e.message}</li>`
        ).join('');
        setStatus(`Validation failed: ${errors.length} error(s)`);
    }
}

function detectCycle(workflow) {
    const edges = {};
    workflow.nodes.forEach(n => edges[n.id] = []);
    workflow.edges.forEach(e => edges[e.source].push(e.target));

    const visited = new Set();
    const recStack = new Set();

    function dfs(node) {
        visited.add(node);
        recStack.add(node);

        for (const next of (edges[node] || [])) {
            if (!visited.has(next)) {
                if (dfs(next)) return true;
            } else if (recStack.has(next)) {
                return true;
            }
        }

        recStack.delete(node);
        return false;
    }

    for (const node of workflow.nodes) {
        if (!visited.has(node.id)) {
            if (dfs(node.id)) return true;
        }
    }
    return false;
}

function exportWorkflow() {
    const workflow = getWorkflowDefinition();
    const blob = new Blob([JSON.stringify(workflow, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'workflow.json';
    a.click();
    URL.revokeObjectURL(url);
    setStatus('Workflow exported');
}

function clearCanvas() {
    state.nodes = [];
    state.edges = [];
    state.selectedNode = null;
    state.nodeCounter = 0;
    canvas.innerHTML = '';
    edgesSvg.innerHTML = '';
    updateOutput();
    setStatus('Canvas cleared');
}

function loadSample() {
    clearCanvas();

    // Create sample workflow
    createNode('trigger_webhook', 100, 150);
    createNode('action_http', 350, 100);
    createNode('logic_condition', 350, 220);
    createNode('action_transform', 600, 100);
    createNode('output_response', 600, 220);
    createNode('output_log', 850, 150);

    // Connect nodes
    setTimeout(() => {
        createEdge('node_1', 'node_2');
        createEdge('node_1', 'node_3');
        createEdge('node_2', 'node_4');
        createEdge('node_3', 'node_5');
        createEdge('node_4', 'node_6');
        createEdge('node_5', 'node_6');
        renderAllEdges();
        setStatus('Sample workflow loaded');
    }, 100);
}

function showTab(tab) {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.querySelector(`.tab[onclick="showTab('${tab}')"]`).classList.add('active');

    document.getElementById('json-output').style.display = tab === 'json' ? 'block' : 'none';
    document.getElementById('dag-output').style.display = tab === 'dag' ? 'block' : 'none';
    document.getElementById('validation-output').style.display = tab === 'validation' ? 'block' : 'none';
    document.getElementById('execution-output').style.display = tab === 'execution' ? 'block' : 'none';
}

// Execution state
const executions = [];

async function simulateExecution(workflow, execution) {
    // Build execution order based on DAG levels
    const edges = {};
    const reverseEdges = {};
    workflow.nodes.forEach(n => {
        edges[n.id] = [];
        reverseEdges[n.id] = [];
    });
    workflow.edges.forEach(e => {
        edges[e.source].push(e.target);
        reverseEdges[e.target].push(e.source);
    });

    // Find entry nodes
    const entryNodes = workflow.nodes.filter(n => reverseEdges[n.id].length === 0).map(n => n.id);

    // Compute levels for parallel execution
    const levels = {};
    const queue = [...entryNodes];
    entryNodes.forEach(id => levels[id] = 0);
    const visited = new Set();

    while (queue.length > 0) {
        const id = queue.shift();
        if (visited.has(id)) continue;
        visited.add(id);

        edges[id].forEach(next => {
            const newLevel = (levels[id] || 0) + 1;
            if (!levels[next] || levels[next] < newLevel) {
                levels[next] = newLevel;
            }
            queue.push(next);
        });
    }

    // Group by level
    const maxLevel = Math.max(0, ...Object.values(levels));

    for (let level = 0; level <= maxLevel; level++) {
        const nodesAtLevel = Object.entries(levels)
            .filter(([_, l]) => l === level)
            .map(([id]) => id);

        // Execute nodes at this level in parallel
        const promises = nodesAtLevel.map(async (nodeId) => {
            const node = workflow.nodes.find(n => n.id === nodeId);

            // Mark as executing
            execution.nodes[nodeId].status = 'running';
            execution.nodes[nodeId].started_at = new Date().toISOString();
            execution.events.push({
                time: new Date().toISOString(),
                message: `Executing: ${node.data.label} (${nodeId})`
            });

            document.getElementById(nodeId)?.classList.add('executing');
            updateExecutionOutput();

            // Simulate execution time based on node type
            const execTime = getNodeExecutionTime(node.type);
            await sleep(execTime);

            // Mark as completed
            execution.nodes[nodeId].status = 'completed';
            execution.nodes[nodeId].finished_at = new Date().toISOString();
            execution.events.push({
                time: new Date().toISOString(),
                message: `Completed: ${node.data.label} (${nodeId}) in ${execTime}ms`
            });

            document.getElementById(nodeId)?.classList.remove('executing');
            document.getElementById(nodeId)?.classList.add('completed');
            updateExecutionOutput();
        });

        await Promise.all(promises);
    }
}

function getNodeExecutionTime(nodeType) {
    const times = {
        trigger_manual: 100,
        trigger_webhook: 150,
        trigger_schedule: 100,
        trigger_event: 120,
        action_http: 500,
        action_script: 300,
        action_transform: 150,
        action_delay: 1000,
        logic_condition: 50,
        logic_switch: 50,
        logic_loop: 200,
        logic_parallel: 50,
        output_response: 100,
        output_log: 50
    };
    return times[nodeType] || 200;
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}

// Enhanced simulation that generates mock node results for visualization
async function simulateExecutionWithResults(workflow, execution) {
    // Build execution order based on DAG levels
    const edges = {};
    const reverseEdges = {};
    workflow.nodes.forEach(n => {
        edges[n.id] = [];
        reverseEdges[n.id] = [];
    });
    workflow.edges.forEach(e => {
        edges[e.source].push(e.target);
        reverseEdges[e.target].push(e.source);
    });

    // Find entry nodes
    const entryNodes = workflow.nodes.filter(n => reverseEdges[n.id].length === 0).map(n => n.id);

    // Compute levels
    const levels = {};
    const queue = [...entryNodes];
    entryNodes.forEach(id => levels[id] = 0);
    const visited = new Set();

    while (queue.length > 0) {
        const id = queue.shift();
        if (visited.has(id)) continue;
        visited.add(id);

        edges[id].forEach(next => {
            const newLevel = (levels[id] || 0) + 1;
            if (!levels[next] || levels[next] < newLevel) {
                levels[next] = newLevel;
            }
            queue.push(next);
        });
    }

    const maxLevel = Math.max(0, ...Object.values(levels));
    const simulatedNodes = [];
    const simulatedLogs = [];
    let sequence = 0;

    for (let level = 0; level <= maxLevel; level++) {
        const nodesAtLevel = Object.entries(levels)
            .filter(([_, l]) => l === level)
            .map(([id]) => id);

        const promises = nodesAtLevel.map(async (nodeId) => {
            const node = workflow.nodes.find(n => n.id === nodeId);
            sequence++;

            // Mark as executing
            execution.nodes[nodeId].status = 'running';
            execution.nodes[nodeId].started_at = new Date().toISOString();

            document.getElementById(nodeId)?.classList.add('executing');

            simulatedLogs.push({
                level: 'info',
                message: `Starting node: ${node.data.label}`,
                logged_at: new Date().toISOString(),
                node_id: nodeId
            });

            updateExecutionOutput();

            const execTime = getNodeExecutionTime(node.type);
            await sleep(execTime);

            // Generate mock output based on node type
            const mockOutput = generateMockOutput(node);

            // Mark as completed
            execution.nodes[nodeId].status = 'completed';
            execution.nodes[nodeId].finished_at = new Date().toISOString();

            simulatedNodes.push({
                node_id: nodeId,
                node_type: node.type,
                status: 'completed',
                sequence: sequence,
                started_at: execution.nodes[nodeId].started_at,
                finished_at: execution.nodes[nodeId].finished_at,
                duration_ms: execTime,
                input: { trigger_data: execution.response?.execution?.trigger_data || {} },
                output: mockOutput
            });

            simulatedLogs.push({
                level: 'info',
                message: `Completed node: ${node.data.label} (${execTime}ms)`,
                logged_at: new Date().toISOString(),
                node_id: nodeId
            });

            document.getElementById(nodeId)?.classList.remove('executing');
            document.getElementById(nodeId)?.classList.add('completed');
            updateExecutionOutput();
        });

        await Promise.all(promises);
    }

    // Store simulated results
    execution.apiNodes = simulatedNodes;
    execution.apiLogs = simulatedLogs;
    execution.events.push({
        time: new Date().toISOString(),
        message: `📊 Simulated: ${simulatedNodes.length} nodes executed`
    });
}

function generateMockOutput(node) {
    switch (node.type) {
        case 'trigger_manual':
            return { triggered: true, timestamp: new Date().toISOString(), source: 'editor' };
        case 'trigger_webhook':
            return { method: 'POST', headers: { 'content-type': 'application/json' }, body: {} };
        case 'trigger_schedule':
            return { scheduled_time: new Date().toISOString(), cron: '0 9 * * *' };
        case 'action_http':
            return {
                status: 200,
                headers: { 'content-type': 'application/json' },
                body: { id: 1, title: 'Sample Response', completed: true }
            };
        case 'action_script':
            return { result: 'Script executed successfully', exit_code: 0 };
        case 'action_transform':
            return { transformed: true, records: 10 };
        case 'action_delay':
            return { delayed_ms: 1000, resumed_at: new Date().toISOString() };
        case 'logic_condition':
            return { condition: 'input.value > 10', result: true, branch: 'true' };
        case 'logic_switch':
            return { matched_case: 'default', value: 'test' };
        case 'logic_loop':
            return { iterations: 5, completed: true };
        case 'logic_parallel':
            return { branches: 2, all_completed: true };
        case 'output_response':
            return { sent: true, status_code: 200 };
        case 'output_log':
            return { logged: true, message: 'Workflow completed' };
        default:
            return { executed: true };
    }
}

function updateExecutionOutput() {
    const container = document.getElementById('execution-output');

    if (executions.length === 0) {
        container.innerHTML = '<p style="color:#64748b;">No executions yet. Click "Execute" to run the workflow.</p>';
        return;
    }

    container.innerHTML = executions.slice(0, 5).map(exec => `
        <div class="exec-log">
            <div class="header">
                <span class="exec-id">${exec.id}</span>
                <span class="status ${exec.status}">${exec.status.toUpperCase()}</span>
            </div>
            <div style="font-size:11px;color:#64748b;margin-bottom:8px;">
                Started: ${new Date(exec.started_at).toLocaleTimeString()}
                ${exec.finished_at ? ' | Finished: ' + new Date(exec.finished_at).toLocaleTimeString() : ''}
                ${exec.api_execution_id ? ` | API ID: ${exec.api_execution_id}` : ''}
            </div>
            <div class="events">
                ${exec.events.slice(-10).map(e => `
                    <div class="event">
                        <span style="color:#64748b;">${new Date(e.time).toLocaleTimeString()}</span>
                        ${e.message}
                    </div>
                `).join('')}
            </div>
            ${exec.api_execution_id ? `
                <button onclick="fetchExecutionDetails(${exec.api_execution_id})" style="margin-top:8px;padding:6px 12px;background:#0f3460;border:1px solid #1a4080;color:#fff;border-radius:4px;cursor:pointer;font-size:11px;">
                    🔄 Fetch Latest Status
                </button>
            ` : ''}
            ${exec.apiNodes && exec.apiNodes.length > 0 ? `
                <details style="margin-top:8px;" open>
                    <summary style="cursor:pointer;color:#f59e0b;">Node Execution Details (${exec.apiNodes.length})</summary>
                    <div style="margin-top:8px;">
                        ${exec.apiNodes.map(node => `
                            <div style="padding:8px;margin:4px 0;background:#0a0f1a;border-radius:4px;border-left:3px solid ${node.status === 'completed' ? '#10b981' : node.status === 'failed' ? '#ef4444' : '#f59e0b'};">
                                <div style="display:flex;justify-content:space-between;">
                                    <span style="color:#fff;">${node.node_id}</span>
                                    <span style="font-size:10px;color:${node.status === 'completed' ? '#10b981' : node.status === 'failed' ? '#ef4444' : '#f59e0b'};">${node.status?.toUpperCase()}</span>
                                </div>
                                ${node.duration_ms ? `<div style="font-size:10px;color:#64748b;">Duration: ${node.duration_ms}ms</div>` : ''}
                                ${node.output ? `<pre style="margin-top:4px;font-size:10px;color:#4ade80;overflow-x:auto;">${JSON.stringify(node.output, null, 2)}</pre>` : ''}
                                ${node.error ? `<div style="margin-top:4px;font-size:10px;color:#ef4444;">${JSON.stringify(node.error)}</div>` : ''}
                            </div>
                        `).join('')}
                    </div>
                </details>
            ` : ''}
            ${exec.apiLogs && exec.apiLogs.length > 0 ? `
                <details style="margin-top:8px;">
                    <summary style="cursor:pointer;color:#3b82f6;">Execution Logs (${exec.apiLogs.length})</summary>
                    <div style="margin-top:8px;max-height:200px;overflow-y:auto;">
                        ${exec.apiLogs.map(log => `
                            <div style="padding:4px 8px;margin:2px 0;background:#0a0f1a;border-radius:2px;font-size:10px;border-left:2px solid ${log.level === 'error' ? '#ef4444' : log.level === 'warning' ? '#f59e0b' : '#64748b'};">
                                <span style="color:#64748b;">${new Date(log.logged_at).toLocaleTimeString()}</span>
                                <span style="color:${log.level === 'error' ? '#ef4444' : log.level === 'warning' ? '#f59e0b' : '#fff'};">[${log.level}]</span>
                                ${log.message}
                            </div>
                        `).join('')}
                    </div>
                </details>
            ` : ''}
            ${exec.response ? `
                <details style="margin-top:8px;">
                    <summary style="cursor:pointer;color:#4ade80;">API Response</summary>
                    <pre style="margin-top:8px;background:#0a0f1a;padding:8px;border-radius:4px;font-size:11px;">${JSON.stringify(exec.response, null, 2)}</pre>
                </details>
            ` : ''}
            ${exec.error ? `<div style="color:#ef4444;margin-top:8px;">Error: ${exec.error}</div>` : ''}
        </div>
    `).join('');
}

async function fetchExecutionDetails(executionId) {
    if (!state.auth.token || !state.auth.currentWorkspace) {
        setStatus('Not logged in');
        return;
    }

    const apiUrl = document.getElementById('apiUrl').value;
    const workspaceId = state.auth.currentWorkspace.id;

    try {
        // Fetch execution details
        const [execResponse, nodesResponse, logsResponse] = await Promise.all([
            fetch(`${apiUrl}/api/v1/workspaces/${workspaceId}/executions/${executionId}`, {
                headers: { 'Accept': 'application/json', 'Authorization': `Bearer ${state.auth.token}` }
            }),
            fetch(`${apiUrl}/api/v1/workspaces/${workspaceId}/executions/${executionId}/nodes`, {
                headers: { 'Accept': 'application/json', 'Authorization': `Bearer ${state.auth.token}` }
            }),
            fetch(`${apiUrl}/api/v1/workspaces/${workspaceId}/executions/${executionId}/logs`, {
                headers: { 'Accept': 'application/json', 'Authorization': `Bearer ${state.auth.token}` }
            })
        ]);

        const execData = await execResponse.json();
        const nodesData = await nodesResponse.json();
        const logsData = await logsResponse.json();

        // Find matching local execution and update it
        const exec = executions.find(e => e.api_execution_id === executionId);
        if (exec) {
            exec.apiExecution = execData.execution || execData.data;
            exec.apiNodes = nodesData.data || nodesData || [];
            exec.apiLogs = logsData.data || logsData || [];
            exec.status = exec.apiExecution?.status || exec.status;

            exec.events.push({
                time: new Date().toISOString(),
                message: `📊 Fetched: ${exec.apiNodes.length} nodes, ${exec.apiLogs.length} logs`
            });

            updateExecutionOutput();
            setStatus(`Updated execution ${executionId}`);
        }
    } catch (err) {
        setStatus(`Failed to fetch: ${err.message}`);
    }
}



function setStatus(msg) {
    document.getElementById('status').textContent = msg;
}

// ==================== AUTH FUNCTIONS ====================

async function login() {
    const apiUrl = document.getElementById('apiUrl').value;
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    const statusEl = document.getElementById('loginStatus');

    statusEl.innerHTML = '<div class="status" style="background:#1a4080;color:#4a9eff;">Logging in...</div>';

    try {
        const response = await fetch(`${apiUrl}/api/v1/auth/login`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            },
            body: JSON.stringify({ email, password })
        });

        const result = await response.json();

        if (response.ok && result.access_token) {
            state.auth.token = result.access_token;
            state.auth.user = result.user;

            statusEl.innerHTML = '<div class="status success">✓ Login successful!</div>';

            // Update UI
            document.getElementById('loginForm').style.display = 'none';
            document.getElementById('userPanel').style.display = 'block';
            document.getElementById('userName').textContent = result.user.first_name + ' ' + result.user.last_name;
            document.getElementById('userEmail').textContent = result.user.email;

            // Load workspaces
            await loadWorkspaces();

            setStatus('✓ Logged in as ' + result.user.email);
        } else {
            throw new Error(result.message || 'Login failed');
        }
    } catch (err) {
        statusEl.innerHTML = `<div class="status error">✗ ${err.message}</div>`;
        setStatus('Login failed: ' + err.message);
    }
}

function logout() {
    state.auth = {
        token: null,
        user: null,
        workspaces: [],
        workflows: [],
        currentWorkspace: null,
        currentWorkflow: null
    };

    document.getElementById('loginForm').style.display = 'block';
    document.getElementById('userPanel').style.display = 'none';
    document.getElementById('loginStatus').innerHTML = '';
    document.getElementById('workspaceSelect').innerHTML = '';
    document.getElementById('workflowSelect').innerHTML = '';

    setStatus('Logged out');
}

async function loadWorkspaces() {
    const apiUrl = document.getElementById('apiUrl').value;

    try {
        const response = await fetch(`${apiUrl}/api/v1/workspaces`, {
            headers: {
                'Accept': 'application/json',
                'Authorization': `Bearer ${state.auth.token}`
            }
        });

        const result = await response.json();

        if (response.ok) {
            state.auth.workspaces = result.data || result.workspaces || [];

            const select = document.getElementById('workspaceSelect');
            select.innerHTML = state.auth.workspaces.map(ws =>
                `<option value="${ws.id}">${ws.name}</option>`
            ).join('');

            if (state.auth.workspaces.length > 0) {
                state.auth.currentWorkspace = state.auth.workspaces[0];
                await loadWorkflows();
            }
        }
    } catch (err) {
        console.error('Failed to load workspaces:', err);
    }
}

async function loadWorkflows() {
    const apiUrl = document.getElementById('apiUrl').value;
    const workspaceId = document.getElementById('workspaceSelect').value;

    if (!workspaceId) return;

    state.auth.currentWorkspace = state.auth.workspaces.find(w => w.id == workspaceId);

    try {
        const response = await fetch(`${apiUrl}/api/v1/workspaces/${workspaceId}/workflows`, {
            headers: {
                'Accept': 'application/json',
                'Authorization': `Bearer ${state.auth.token}`
            }
        });

        const result = await response.json();

        if (response.ok) {
            state.auth.workflows = result.data || result.workflows || [];

            const select = document.getElementById('workflowSelect');
            select.innerHTML = '<option value="">-- Select workflow --</option>' +
                state.auth.workflows.map(wf =>
                    `<option value="${wf.id}">${wf.name} ${wf.is_active ? '✓' : ''}</option>`
                ).join('');
        }
    } catch (err) {
        console.error('Failed to load workflows:', err);
    }
}

function loadWorkflowToCanvas() {
    const workflowId = document.getElementById('workflowSelect').value;
    if (!workflowId) return;

    const workflow = state.auth.workflows.find(w => w.id == workflowId);
    if (!workflow) return;

    state.auth.currentWorkflow = workflow;

    // Clear canvas
    clearCanvas();

    // Load nodes
    if (workflow.nodes && Array.isArray(workflow.nodes)) {
        workflow.nodes.forEach((node, idx) => {
            state.nodeCounter = Math.max(state.nodeCounter, idx + 1);
            state.nodes.push({
                id: node.id,
                type: node.type,
                position: node.position || { x: 100 + idx * 250, y: 150 },
                data: node.data || { label: node.type, config: {} }
            });
        });

        state.nodes.forEach(node => renderNode(node));
    }

    // Load edges
    if (workflow.edges && Array.isArray(workflow.edges)) {
        workflow.edges.forEach(edge => {
            state.edges.push({
                id: edge.id,
                source: edge.source,
                target: edge.target,
                sourceHandle: 'output',
                targetHandle: 'input'
            });
        });

        setTimeout(() => renderAllEdges(), 100);
    }

    updateOutput();
    setStatus(`Loaded workflow: ${workflow.name}`);
}

async function executeWorkflow() {
    const workflow = getWorkflowDefinition();

    // Validate first
    const triggers = workflow.nodes.filter(n => n.type.startsWith('trigger_'));
    if (triggers.length === 0) {
        setStatus('Error: Workflow needs a trigger node');
        return;
    }

    if (detectCycle(workflow)) {
        setStatus('Error: Workflow has cycles');
        return;
    }

    // Check if logged in and workflow selected
    const useApi = state.auth.token && state.auth.currentWorkflow;

    setStatus(useApi ? 'Executing workflow via API...' : 'Simulating workflow...');
    showTab('execution');

    // Reset node states
    document.querySelectorAll('.workflow-node').forEach(n => {
        n.classList.remove('executing', 'completed', 'failed');
    });

    const execution = {
        id: (useApi ? 'api-' : 'sim-') + Date.now(),
        workflow_id: useApi ? state.auth.currentWorkflow.id : workflow.id,
        status: 'pending',
        started_at: new Date().toISOString(),
        nodes: {},
        events: [],
        response: null,
        error: null
    };

    // Initialize node states
    workflow.nodes.forEach(n => {
        execution.nodes[n.id] = { status: 'pending', started_at: null, finished_at: null };
    });

    executions.unshift(execution);
    updateExecutionOutput();

    if (useApi) {
        // Call real API
        try {
            const apiUrl = document.getElementById('apiUrl').value;
            const workspaceId = state.auth.currentWorkspace.id;
            const workflowId = state.auth.currentWorkflow.id;

            execution.status = 'running';
            execution.events.push({ time: new Date().toISOString(), message: '🚀 Calling API...' });
            updateExecutionOutput();

            const response = await fetch(`${apiUrl}/api/v1/workspaces/${workspaceId}/workflows/${workflowId}/execute`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Accept': 'application/json',
                    'Authorization': `Bearer ${state.auth.token}`
                },
                body: JSON.stringify({
                    input: {
                        workflow_definition: workflow,
                        trigger_data: { manual: true, timestamp: new Date().toISOString() }
                    }
                })
            });

            const result = await response.json();

            if (response.ok) {
                execution.response = result;
                execution.api_execution_id = result.execution?.id;
                execution.events.push({
                    time: new Date().toISOString(),
                    message: `✓ API Response: execution_id=${result.execution?.id || 'created'}`
                });

                // Simulate visual execution with mock node results
                await simulateExecutionWithResults(workflow, execution);

                execution.status = 'completed';
                execution.finished_at = new Date().toISOString();
                execution.events.push({ time: new Date().toISOString(), message: '✅ Execution queued on API - Go engine processing' });
                setStatus('Workflow executed! Simulated node results shown below.');

                // Auto-fetch after a delay (may be empty if Go engine hasn't completed)
                setTimeout(() => fetchExecutionDetails(execution.api_execution_id), 3000);
            } else {
                throw new Error(result.message || 'API error');
            }
        } catch (err) {
            execution.events.push({ time: new Date().toISOString(), message: `❌ Error: ${err.message}` });
            execution.status = 'failed';
            execution.error = err.message;
            execution.finished_at = new Date().toISOString();
            setStatus(`Execution failed: ${err.message}`);
        }
    } else {
        // Local simulation only
        execution.status = 'running';
        execution.events.push({ time: new Date().toISOString(), message: '🧪 Running local simulation...' });
        updateExecutionOutput();

        await simulateExecution(workflow, execution);

        execution.status = 'completed';
        execution.finished_at = new Date().toISOString();
        execution.events.push({ time: new Date().toISOString(), message: '✅ Simulation completed' });
        setStatus('Simulation completed');
    }

    updateExecutionOutput();
}

// Initialize
updateOutput();
// Check for saved token
const savedToken = localStorage.getItem('lnkflow_token');
if (savedToken) {
    state.auth.token = savedToken;
    // Could auto-login here if needed
}
