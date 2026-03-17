import { useCallback, useEffect, useMemo, useState } from 'react';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  type NodeTypes,
  Handle,
  Position,
} from 'reactflow';
import 'reactflow/dist/style.css';
import type { GraphData, GraphNode } from '../types';

// --- Color palette for categories ---
const PALETTE = [
  '#6366f1', '#22c55e', '#f59e0b', '#ef4444',
  '#ec4899', '#06b6d4', '#8b5cf6', '#14b8a6',
];

function getCategoryColor(cat: string, index: number): string {
  // Deterministic color from palette based on hash
  let hash = 0;
  for (let i = 0; i < cat.length; i++) hash = cat.charCodeAt(i) + ((hash << 5) - hash);
  return PALETTE[Math.abs(hash) % PALETTE.length] ?? PALETTE[index % PALETTE.length];
}

// --- Custom node components ---
interface CategoryNodeData {
  label: string;
  color: string;
}

function CategoryNode({ data }: { data: CategoryNodeData }) {
  return (
    <div style={{
      padding: '12px 20px',
      borderRadius: 10,
      background: data.color + '22',
      border: `2px solid ${data.color}`,
      color: data.color,
      fontWeight: 700,
      fontSize: 13,
      minWidth: 100,
      textAlign: 'center',
    }}>
      <Handle type="source" position={Position.Bottom} style={{ background: data.color }} />
      {data.label}
    </div>
  );
}

interface LeafNodeData {
  label: string;
  nodeType: 'memory' | 'note';
  details: Record<string, string>;
}

function LeafNode({ data }: { data: LeafNodeData }) {
  const color = data.nodeType === 'memory' ? '#6366f1' : '#22c55e';
  return (
    <div style={{
      padding: '8px 14px',
      borderRadius: 8,
      background: 'var(--bg-card)',
      border: `1px solid ${color}66`,
      color: 'var(--text)',
      fontSize: 12,
      maxWidth: 180,
      wordBreak: 'break-word',
    }}>
      <Handle type="target" position={Position.Top} style={{ background: color }} />
      <div style={{ fontSize: 10, color, fontWeight: 700, textTransform: 'uppercase', marginBottom: 2 }}>
        {data.nodeType}
      </div>
      <div style={{ fontFamily: data.nodeType === 'memory' ? 'var(--font-mono)' : 'inherit' }}>
        {data.label.length > 40 ? data.label.slice(0, 40) + '…' : data.label}
      </div>
    </div>
  );
}

const nodeTypes: NodeTypes = {
  category: CategoryNode as NodeTypes['category'],
  leaf: LeafNode as NodeTypes['leaf'],
};

// --- Layout: simple radial / grid ---
function buildFlowNodes(
  apiNodes: GraphNode[],
): { flowNodes: Node[]; flowEdges: Edge[] } {
  const categories = apiNodes.filter(n => n.type === 'category');
  const leafNodes = apiNodes.filter(n => n.type !== 'category');

  const catColorMap: Record<string, string> = {};
  categories.forEach((c, i) => {
    catColorMap[c.id] = getCategoryColor(c.label, i);
  });

  // Position categories in a horizontal row at top
  const CAT_GAP = 260;
  const CAT_Y = 0;
  const catPositions: Record<string, { x: number; y: number }> = {};
  categories.forEach((c, i) => {
    const x = i * CAT_GAP - ((categories.length - 1) * CAT_GAP) / 2;
    catPositions[c.id] = { x, y: CAT_Y };
  });

  // Position leaf nodes below their parent category
  const leafByParent: Record<string, GraphNode[]> = {};
  leafNodes.forEach(n => {
    // We need edge info to find parent — we'll match by id prefix convention
    // cat:X -> mem:N or note:N
    // We'll find parent category by looking at edges later; for now use data.category
    const catId = n.type === 'memory'
      ? 'cat:' + (n.data.category ?? '')
      : 'cat:notes' + (n.data.project ? ':' + n.data.project : '');
    if (!leafByParent[catId]) leafByParent[catId] = [];
    leafByParent[catId].push(n);
  });

  const LEAF_GAP_X = 200;
  const LEAF_GAP_Y = 90;
  const LEAF_START_Y = 140;

  const flowNodes: Node[] = [];

  categories.forEach(c => {
    const pos = catPositions[c.id] ?? { x: 0, y: 0 };
    flowNodes.push({
      id: c.id,
      type: 'category',
      position: pos,
      data: { label: c.label, color: catColorMap[c.id] ?? '#6366f1' },
    });

    const children = leafByParent[c.id] ?? [];
    const cols = 3;
    children.forEach((child, i) => {
      const col = i % cols;
      const row = Math.floor(i / cols);
      flowNodes.push({
        id: child.id,
        type: 'leaf',
        position: {
          x: pos.x + (col - Math.floor(cols / 2)) * LEAF_GAP_X,
          y: LEAF_START_Y + row * LEAF_GAP_Y,
        },
        data: {
          label: child.label,
          nodeType: child.type as 'memory' | 'note',
          details: child.data,
        },
      });
    });
  });

  return { flowNodes, flowEdges: [] };
}

// --- Detail panel ---
interface DetailPanelProps {
  node: Node | null;
  onClose: () => void;
}

function DetailPanel({ node, onClose }: DetailPanelProps) {
  if (!node) return null;
  const data = node.data as Record<string, unknown>;
  const details = (data.details ?? {}) as Record<string, string>;

  return (
    <div style={{
      position: 'absolute',
      top: 16,
      right: 16,
      width: 300,
      background: 'var(--bg-card)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius)',
      padding: 16,
      zIndex: 10,
      boxShadow: '0 4px 24px rgba(0,0,0,0.4)',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <span style={{ fontWeight: 600, color: 'var(--text)' }}>
          {String(data.label ?? '')}
        </span>
        <button
          onClick={onClose}
          style={{
            background: 'none',
            border: 'none',
            color: 'var(--text-muted)',
            fontSize: 18,
            lineHeight: 1,
            padding: '0 4px',
          }}
        >
          ×
        </button>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {Object.entries(details).map(([k, v]) => (
          <div key={k}>
            <div style={{ fontSize: 11, color: 'var(--text-dim)', textTransform: 'uppercase', marginBottom: 2 }}>
              {k}
            </div>
            <div style={{
              fontSize: 13, color: 'var(--text-muted)',
              wordBreak: 'break-word',
              maxHeight: 120,
              overflowY: 'auto',
            }}>
              {v || '—'}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// --- Main component ---
interface KnowledgeGraphProps {
  graphData: GraphData | null;
  loading: boolean;
  error: string | null;
}

export function KnowledgeGraph({ graphData, loading, error }: KnowledgeGraphProps) {
  const { flowNodes: initialNodes, flowEdges: initialEdgesFromLayout } = useMemo(() => {
    if (!graphData) return { flowNodes: [], flowEdges: [] };
    return buildFlowNodes(graphData.nodes);
  }, [graphData]);

  // Build actual edges from API edge data
  const initialEdges: Edge[] = useMemo(() => {
    if (!graphData) return [];
    return graphData.edges.map(e => ({
      id: e.id,
      source: e.source,
      target: e.target,
      style: { stroke: '#334155', strokeWidth: 1 },
      animated: false,
    }));
  }, [graphData]);

  // Suppress unused warning
  void initialEdgesFromLayout;

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);

  // Sync nodes/edges when graphData changes (useNodesState doesn't re-init on prop change).
  useEffect(() => { setNodes(initialNodes); }, [initialNodes, setNodes]);
  useEffect(() => { setEdges(initialEdges); }, [initialEdges, setEdges]);

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);

  if (loading) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-muted)' }}>
        Loading graph data…
      </div>
    );
  }
  if (error) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--error)' }}>
        {error}
      </div>
    );
  }
  if (!graphData || graphData.nodes.length === 0) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--text-dim)' }}>
        No data available. Add some memories or notes to see the graph.
      </div>
    );
  }

  return (
    <div style={{ position: 'relative', width: '100%', height: '100%' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={onNodeClick}
        nodeTypes={nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.2 }}
      >
        <Background color="#1e293b" gap={24} />
        <Controls style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }} />
        <MiniMap
          style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
          nodeColor={(n) => {
            const d = n.data as Record<string, unknown>;
            return typeof d.color === 'string' ? d.color : '#334155';
          }}
        />
      </ReactFlow>
      <DetailPanel node={selectedNode} onClose={() => setSelectedNode(null)} />
    </div>
  );
}
