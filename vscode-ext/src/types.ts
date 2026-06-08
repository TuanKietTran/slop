export interface Task {
  id: string
  target_site: string
  status: string
  stage: string
  skills: string[]
  providers: Record<string, string>
  created_at: string
  updated_at: string
}
