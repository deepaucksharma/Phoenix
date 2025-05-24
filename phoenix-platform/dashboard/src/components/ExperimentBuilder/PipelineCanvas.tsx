import React, { useCallback, useState, useEffect } from 'react'
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  ReactFlowProvider,
  Panel,
} from 'react-flow-renderer'
import { Box, Paper, Button, IconButton, Tooltip } from '@mui/material'
import {
  Save as SaveIcon,
  PlayArrow as RunIcon,
  Clear as ClearIcon,
  Download as ExportIcon,
} from '@mui/icons-material'

import { ProcessorNode } from './ProcessorNode'
import { ConfigurationPanel } from './ConfigurationPanel'
import { usePipelineGenerator } from '@/hooks/usePipelineGenerator'
import { ProcessorNodeData, PipelineConfig } from '@/types/pipeline'

const nodeTypes = {
  processor: ProcessorNode,
}

interface PipelineCanvasProps {
  initialConfig?: PipelineConfig
  onSave?: (config: PipelineConfig) => void
  onRun?: (config: PipelineConfig) => void
}

export const PipelineCanvas: React.FC<PipelineCanvasProps> = ({
  initialConfig,
  onSave,
  onRun,
}) => {
  const [nodes, setNodes, onNodesChange] = useNodesState(
    initialConfig?.nodes || []
  )
  const [edges, setEdges, onEdgesChange] = useEdgesState(
    initialConfig?.edges || []
  )
  const [selectedNode, setSelectedNode] = useState<Node<ProcessorNodeData> | null>(
    null
  )
  
  const { generateConfig, validatePipeline } = usePipelineGenerator()

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => addEdge({ ...params, type: 'smoothstep' }, eds))
    },
    [setEdges]
  )

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault()

      const reactFlowBounds = event.currentTarget.getBoundingClientRect()
      const type = event.dataTransfer.getData('application/reactflow')

      if (!type) return

      const position = {
        x: event.clientX - reactFlowBounds.left,
        y: event.clientY - reactFlowBounds.top,
      }

      const newNode: Node<ProcessorNodeData> = {
        id: `${type}_${Date.now()}`,
        type: 'processor',
        position,
        data: {
          label: type,
          processorType: type as any,
          config: {},
        },
      }

      setNodes((nds) => nds.concat(newNode))
    },
    [setNodes]
  )

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault()
    event.dataTransfer.dropEffect = 'move'
  }, [])

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNode(node as Node<ProcessorNodeData>)
  }, [])

  const handleNodeUpdate = useCallback(
    (nodeId: string, data: ProcessorNodeData) => {
      setNodes((nds) =>
        nds.map((node) => {
          if (node.id === nodeId) {
            return { ...node, data }
          }
          return node
        })
      )
    },
    [setNodes]
  )

  const handleSave = useCallback(() => {
    const config: PipelineConfig = {
      nodes,
      edges,
      metadata: {
        createdAt: new Date().toISOString(),
        version: '1.0.0',
      },
    }

    const validation = validatePipeline(config)
    if (!validation.valid) {
      console.error('Pipeline validation failed:', validation.errors)
      return
    }

    onSave?.(config)
  }, [nodes, edges, validatePipeline, onSave])

  const handleRun = useCallback(() => {
    const config: PipelineConfig = { nodes, edges }
    onRun?.(config)
  }, [nodes, edges, onRun])

  const handleClear = useCallback(() => {
    setNodes([])
    setEdges([])
    setSelectedNode(null)
  }, [setNodes, setEdges])

  const handleExport = useCallback(() => {
    const config = generateConfig(nodes, edges)
    const blob = new Blob([config], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'pipeline-config.yaml'
    a.click()
    URL.revokeObjectURL(url)
  }, [nodes, edges, generateConfig])

  return (
    <Box sx={{ display: 'flex', height: '100%', width: '100%' }}>
      <Paper
        sx={{
          flex: 1,
          position: 'relative',
          backgroundColor: '#fafafa',
        }}
        elevation={0}
      >
        <ReactFlowProvider>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onDrop={onDrop}
            onDragOver={onDragOver}
            onNodeClick={onNodeClick}
            nodeTypes={nodeTypes}
            fitView
          >
            <Panel position="top-left">
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Tooltip title="Save Pipeline">
                  <IconButton onClick={handleSave} color="primary">
                    <SaveIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Run Experiment">
                  <IconButton onClick={handleRun} color="success">
                    <RunIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Export YAML">
                  <IconButton onClick={handleExport}>
                    <ExportIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Clear Canvas">
                  <IconButton onClick={handleClear} color="error">
                    <ClearIcon />
                  </IconButton>
                </Tooltip>
              </Box>
            </Panel>
            <Controls />
            <MiniMap />
            <Background variant="dots" gap={12} size={1} />
          </ReactFlow>
        </ReactFlowProvider>
      </Paper>
      
      {selectedNode && (
        <ConfigurationPanel
          node={selectedNode}
          onUpdate={handleNodeUpdate}
          onClose={() => setSelectedNode(null)}
        />
      )}
    </Box>
  )
}