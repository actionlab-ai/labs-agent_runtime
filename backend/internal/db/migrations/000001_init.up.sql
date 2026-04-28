-- 创建项目表 (projects)
-- 存储项目的核心信息，支持软删除（通过 deleted_at 字段）
CREATE TABLE IF NOT EXISTS projects (
  id text PRIMARY KEY, -- 项目唯一标识符
  name text NOT NULL, -- 项目名称
  description text NOT NULL DEFAULT '', -- 项目描述
  status text NOT NULL DEFAULT 'active', -- 项目状态（如：active, archived）
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb, -- 扩展元数据，使用 JSONB 格式存储
  created_at timestamptz NOT NULL DEFAULT now(), -- 创建时间
  updated_at timestamptz NOT NULL DEFAULT now(), -- 最后更新时间
  deleted_at timestamptz -- 软删除时间，若为 NULL 则表示未删除
);

-- 创建项目文档表 (project_documents)
-- 存储与项目关联的文档内容，支持多种文档类型（kind）
CREATE TABLE IF NOT EXISTS project_documents (
  id bigserial PRIMARY KEY, -- 自增主键
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE, -- 关联的项目 ID，项目删除时级联删除文档
  kind text NOT NULL, -- 文档类型（例如：spec, design, code）
  title text NOT NULL, -- 文档标题
  body text NOT NULL DEFAULT '', -- 文档正文内容
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb, -- 扩展元数据
  created_at timestamptz NOT NULL DEFAULT now(), -- 创建时间
  updated_at timestamptz NOT NULL DEFAULT now(), -- 最后更新时间
  deleted_at timestamptz -- 软删除时间
);

-- 创建唯一索引：确保每个项目在每种文档类型下只有一个有效的（未删除的）文档
CREATE UNIQUE INDEX IF NOT EXISTS project_documents_active_kind_uidx
  ON project_documents(project_id, kind)
  WHERE deleted_at IS NULL;

-- 创建会话表 (sessions)
-- 存储用户与 AI 交互的会话记录
CREATE TABLE IF NOT EXISTS sessions (
  id bigserial PRIMARY KEY, -- 自增主键
  project_id text REFERENCES projects(id) ON DELETE SET NULL, -- 关联的项目 ID，项目删除时设为 NULL
  title text NOT NULL DEFAULT '', -- 会话标题
  status text NOT NULL DEFAULT 'active', -- 会话状态
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb, -- 扩展元数据
  created_at timestamptz NOT NULL DEFAULT now(), -- 创建时间
  updated_at timestamptz NOT NULL DEFAULT now(), -- 最后更新时间
  deleted_at timestamptz -- 软删除时间
);

-- 创建消息表 (messages)
-- 存储会话中的具体对话消息（用户输入或 AI 回复）
CREATE TABLE IF NOT EXISTS messages (
  id bigserial PRIMARY KEY, -- 自增主键
  session_id bigint NOT NULL REFERENCES sessions(id) ON DELETE CASCADE, -- 关联的会话 ID，会话删除时级联删除消息
  role text NOT NULL, -- 消息角色（例如：user, assistant, system）
  content text NOT NULL DEFAULT '', -- 消息内容
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb, -- 扩展元数据
  created_at timestamptz NOT NULL DEFAULT now() -- 创建时间
);

-- 创建运行记录表 (runs)
-- 存储每次 AI 执行任务或生成内容的详细运行记录
CREATE TABLE IF NOT EXISTS runs (
  id bigserial PRIMARY KEY, -- 自增主键
  project_id text REFERENCES projects(id) ON DELETE SET NULL, -- 关联的项目 ID
  session_id bigint REFERENCES sessions(id) ON DELETE SET NULL, -- 关联的会话 ID
  input text NOT NULL, -- 用户输入的提示词或指令
  final_text text NOT NULL DEFAULT '', -- AI 生成的最终文本结果
  run_dir text NOT NULL DEFAULT '', -- 运行时的目录路径
  status text NOT NULL DEFAULT 'running', -- 运行状态（例如：running, completed, failed）
  error text NOT NULL DEFAULT '', -- 错误信息（如果有）
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb, -- 扩展元数据
  created_at timestamptz NOT NULL DEFAULT now(), -- 创建时间
  updated_at timestamptz NOT NULL DEFAULT now() -- 最后更新时间
);

-- 创建或替换函数：set_updated_at()
-- 该函数用于触发器中，自动将 updated_at 字段设置为当前时间
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now(); -- 更新 new 记录的 updated_at 为当前时间
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为 projects 表创建触发器：在更新前自动调用 set_updated_at()
DROP TRIGGER IF EXISTS projects_set_updated_at ON projects;
CREATE TRIGGER projects_set_updated_at
  BEFORE UPDATE ON projects
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 为 project_documents 表创建触发器：在更新前自动调用 set_updated_at()
DROP TRIGGER IF EXISTS project_documents_set_updated_at ON project_documents;
CREATE TRIGGER project_documents_set_updated_at
  BEFORE UPDATE ON project_documents
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 为 sessions 表创建触发器：在更新前自动调用 set_updated_at()
DROP TRIGGER IF EXISTS sessions_set_updated_at ON sessions;
CREATE TRIGGER sessions_set_updated_at
  BEFORE UPDATE ON sessions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 为 runs 表创建触发器：在更新前自动调用 set_updated_at()
DROP TRIGGER IF EXISTS runs_set_updated_at ON runs;
CREATE TRIGGER runs_set_updated_at
  BEFORE UPDATE ON runs
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();