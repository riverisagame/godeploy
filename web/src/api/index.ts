import request from './request'
import type { Project, Environment, Deployment, Server } from '../types'

export const api = {
  // Auth
  login: (data: any) => request.post('/login', data),

  // Users
  getUsers: () => request.get('/users'),
  createUser: (data: any) => request.post('/users', data),

  // Projects
  getProjects: () => request.get<Project[]>('/projects'),
  createProject: (data: Partial<Project>) => request.post<Project>('/projects', data),
  updateProject: (id: number | string, data: Partial<Project>) => request.put<Project>(`/projects/${id}`, data),
  deleteProject: (id: number | string) => request.delete(`/projects/${id}`),
  
  // Environments
  getProjectEnvironments: (projectId: string | number) => request.get<Environment[]>(`/projects/${projectId}/environments`),
  createEnvironment: (projectId: string | number, data: Partial<Environment>) => request.post<Project>(`/projects/${projectId}/environments`, data),
  updateEnvironment: (projectId: string | number, envName: string, data: Partial<Environment>) => request.put<Environment>(`/projects/${projectId}/environments/${envName}`, data),
  getEnvironmentDiff: (projectId: string | number, envName: string) => request.get(`/projects/${projectId}/environments/${envName}/diff`),

  // Deployments
  getDeployments: (envId: number) => request.get<Deployment[]>(`/deployments`, { params: { env_id: envId } }),
  createDeployment: (projectId: string | number, envName: string, branch: string) => request.post<Deployment>('/deployments', { project_id: typeof projectId === 'string' ? parseInt(projectId) : projectId, env_name: envName, branch }),
  rollbackDeployment: (deployId: number, projectId: string | number, envName: string, targetRelease: string) => request.post<Deployment>(`/deployments/${deployId}/rollback`, { project_id: typeof projectId === 'string' ? parseInt(projectId) : projectId, env_name: envName, target_release: targetRelease }),
  cancelDeployment: (deployId: number) => request.post<any>(`/deployments/${deployId}/cancel`),
  
  // Servers
  getServers: () => request.get<Server[]>('/servers'),
  createServer: (data: Partial<Server>) => request.post<Server>('/servers', data),
  updateServer: (id: number | string, data: Partial<Server>) => request.put<Server>(`/servers/${id}`, data),
  deleteServer: (id: number | string) => request.delete(`/servers/${id}`),

  // Audit Logs
  getAuditLogs: (page: number, pageSize: number) => request.get('/audit-logs', { params: { page, pageSize } }),
}
