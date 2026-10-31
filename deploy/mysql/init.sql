-- 仅负责建库与字符集；表结构由 Panel 启动时 GORM AutoMigrate 创建。
CREATE DATABASE IF NOT EXISTS aoe_tunnel
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

USE aoe_tunnel;
