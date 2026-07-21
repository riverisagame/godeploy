export interface EnvVar {
  key: string
  value: string
  is_secret?: boolean
}

export interface CommitInfo {
  hash: string
  message: string
  author: string
  date: string
}

export interface Deployment {
  id: number
  env_id: number
  user_id: number
  commit_hash: string
  status: string
  phase: string
  log: string
  release_name: string
  created_at: string
}

export interface Environment {
  id: number
  name: string
  branch: string
  deploy_type: string
  build_command: string
  pre_deploy: string
  post_deploy: string
  shared_dirs: string
  shared_files: string
  server_ids: number[]
  deploy_path: string
  env_vars: EnvVar[]
  deployments?: Deployment[] // Frontend logic adds this
}

export interface Project {
  id: number
  name: string
  repo_url: string
  keep_releases: number
  environments: Environment[]
}

export interface Server {
  id: number
  name: string
  ip: string
  port: number
  user: string
  key_path: string
}
