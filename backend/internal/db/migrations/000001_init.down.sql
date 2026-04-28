-- 删除用于自动更新 'updated_at' 时间戳的触发器
DROP TRIGGER IF EXISTS runs_set_updated_at ON runs;
DROP TRIGGER IF EXISTS sessions_set_updated_at ON sessions;
DROP TRIGGER IF EXISTS project_documents_set_updated_at ON project_documents;
DROP TRIGGER IF EXISTS projects_set_updated_at ON projects;

-- 删除上述触发器所依赖的通用函数
DROP FUNCTION IF EXISTS set_updated_at();

-- 按照依赖关系的逆序删除数据表，以避免外键约束错误
-- 1. 删除依赖会话和消息的运行记录表
DROP TABLE IF EXISTS runs;
-- 2. 删除依赖会话的消息表
DROP TABLE IF EXISTS messages;
-- 3. 删除依赖项目的会话表
DROP TABLE IF EXISTS sessions;
-- 4. 删除依赖项目的项目文档表
DROP TABLE IF EXISTS project_documents;
-- 5. 最后删除基础的项目表
DROP TABLE IF EXISTS projects;