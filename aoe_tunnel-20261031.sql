-- MySQL dump 10.13  Distrib 8.0.46, for Linux (x86_64)
--
-- Host: localhost    Database: aoe_tunnel
-- ------------------------------------------------------
-- Server version	8.0.46

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Current Database: `aoe_tunnel`
--

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `aoe_tunnel` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `aoe_tunnel`;

--
-- Table structure for table `api_keys`
--

DROP TABLE IF EXISTS `api_keys`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `api_keys` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `name` varchar(64) NOT NULL,
  `key_prefix` varchar(16) NOT NULL,
  `key_hash` varchar(64) NOT NULL,
  `scope_codes` json DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `expires_at` datetime(3) DEFAULT NULL,
  `last_used_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_api_keys_key_hash` (`key_hash`),
  KEY `idx_api_keys_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `api_keys`
--

LOCK TABLES `api_keys` WRITE;
/*!40000 ALTER TABLE `api_keys` DISABLE KEYS */;
INSERT INTO `api_keys` VALUES (1,1,'it4-agent','aoe_sk_a36bbb3d','a8be92ca6fc4a5d582f81e045bf1ed6679ab36cda2f1703d1efb75a740434cd6','[]',1,NULL,'2026-10-30 22:40:30.445','2026-10-30 22:40:30.442','2026-10-30 22:40:30.445'),(2,1,'it4-agent','aoe_sk_2b8e2933','80d785530b8cb0d434af0eaa1bc1ecea34bdc9133668b179f269b25887098522','[]',1,NULL,'2026-10-30 22:41:35.301','2026-10-30 22:41:35.298','2026-10-30 22:43:07.563');
/*!40000 ALTER TABLE `api_keys` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `commission_records`
--

DROP TABLE IF EXISTS `commission_records`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `commission_records` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `inviter_id` bigint unsigned NOT NULL,
  `buyer_id` bigint unsigned NOT NULL,
  `package_id` bigint unsigned DEFAULT NULL,
  `amount` decimal(14,2) NOT NULL DEFAULT '0.00',
  `ratio` decimal(6,2) NOT NULL DEFAULT '0.00',
  `commission` decimal(14,2) NOT NULL DEFAULT '0.00',
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_commission_records_inviter_id` (`inviter_id`),
  KEY `idx_commission_records_buyer_id` (`buyer_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `commission_records`
--

LOCK TABLES `commission_records` WRITE;
/*!40000 ALTER TABLE `commission_records` DISABLE KEYS */;
INSERT INTO `commission_records` VALUES (1,1,12,12,100.00,20.00,20.00,'2026-10-31 15:10:49.137');
/*!40000 ALTER TABLE `commission_records` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `node_instances`
--

DROP TABLE IF EXISTS `node_instances`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_instances` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `node_code` varchar(64) NOT NULL,
  `instance_id` varchar(64) NOT NULL,
  `ip` varchar(64) NOT NULL DEFAULT '',
  `quic_addr` varchar(128) NOT NULL DEFAULT '',
  `tls_addr` varchar(128) NOT NULL DEFAULT '',
  `version` varchar(32) NOT NULL DEFAULT '',
  `online` tinyint(1) NOT NULL DEFAULT '0',
  `last_seen_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_node_instance` (`node_code`,`instance_id`),
  KEY `idx_node_instances_node_code` (`node_code`)
) ENGINE=InnoDB AUTO_INCREMENT=1091 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `node_instances`
--

LOCK TABLES `node_instances` WRITE;
/*!40000 ALTER TABLE `node_instances` DISABLE KEYS */;
INSERT INTO `node_instances` VALUES (719,'n-5d956216','135a0be9-3732-47ba-80ab-1bb600beafcb','129.226.185.231','129.226.185.231:9703','129.226.185.231:9704','0.1.0',0,'2026-10-31 14:38:40.739','2026-10-31 14:38:40.740','2026-10-31 14:38:55.441'),(720,'n-5dfb85fe','38f04a64-da7b-4b1b-acf1-1bbc6fa8e4d6','132.232.177.36','132.232.177.36:9443','132.232.177.36:9444','0.1.0',1,'2026-10-31 16:15:01.211','2026-10-31 14:41:31.065','2026-10-31 16:15:01.211');
/*!40000 ALTER TABLE `node_instances` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `nodes`
--

DROP TABLE IF EXISTS `nodes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nodes` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `name` varchar(128) NOT NULL,
  `quic_addr` varchar(128) NOT NULL,
  `tls_addr` varchar(128) NOT NULL,
  `version` varchar(32) DEFAULT NULL,
  `status` varchar(32) NOT NULL DEFAULT 'offline',
  `last_seen_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `type` bigint NOT NULL DEFAULT '0',
  `rev_port` bigint NOT NULL DEFAULT '0',
  `rev_group` varchar(64) NOT NULL DEFAULT '',
  `node_token_hash` varchar(64) NOT NULL DEFAULT '',
  `node_token_epoch` bigint NOT NULL DEFAULT '1',
  `ip` varchar(64) NOT NULL DEFAULT '',
  `rate_up_m_bps` decimal(10,3) NOT NULL DEFAULT '0.000',
  `rate_down_m_bps` decimal(10,3) NOT NULL DEFAULT '0.000',
  `rate_up_mbps` decimal(10,3) NOT NULL DEFAULT '0.000',
  `rate_down_mbps` decimal(10,3) NOT NULL DEFAULT '0.000',
  `ratio_up` decimal(6,3) NOT NULL DEFAULT '1.000',
  `ratio_down` decimal(6,3) NOT NULL DEFAULT '1.000',
  `public_addr` varchar(255) NOT NULL DEFAULT '',
  `active_instance` varchar(64) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_nodes_code` (`code`),
  KEY `idx_nodes_type` (`type`),
  KEY `idx_nodes_rev_group` (`rev_group`)
) ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `nodes`
--

LOCK TABLES `nodes` WRITE;
/*!40000 ALTER TABLE `nodes` DISABLE KEYS */;
INSERT INTO `nodes` VALUES (17,'n-5dfb85fe','测试','132.232.177.36:9443','132.232.177.36:9444','0.1.0','online','2026-10-31 16:15:08.780','2026-10-31 14:41:01.530','2026-10-31 16:15:08.780',1,0,'','b5a01a8b6c395ccf239c0f2f3880f0cc299e88249748bc0e69ebd395964cf3db',1,'132.232.177.36',0.000,0.000,0.000,0.000,1.000,1.000,'','38f04a64-da7b-4b1b-acf1-1bbc6fa8e4d6'),(18,'n-497f6d3e','出口测试','','','','offline',NULL,'2026-10-31 14:55:26.329','2026-10-31 14:55:26.329',2,0,'','6d27887c9f43f72d4573b8b18946567ceb5c759009fab6e09250b1680a97bb3e',1,'',0.000,0.000,0.000,0.000,1.000,1.000,'','');
/*!40000 ALTER TABLE `nodes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `packages`
--

DROP TABLE IF EXISTS `packages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `packages` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `ingress_nodes` json DEFAULT NULL,
  `egress_nodes` json DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `traffic_limit_gb` decimal(14,3) NOT NULL DEFAULT '0.000',
  `rule_limit` bigint NOT NULL DEFAULT '0',
  `price` decimal(14,2) NOT NULL DEFAULT '0.00',
  `duration` varchar(16) NOT NULL DEFAULT 'monthly',
  `prices` json DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `packages`
--

LOCK TABLES `packages` WRITE;
/*!40000 ALTER TABLE `packages` DISABLE KEYS */;
INSERT INTO `packages` VALUES (12,'测试套餐','','[\"n-5dfb85fe\"]','[\"n-497f6d3e\"]',1,'2026-10-31 14:43:41.161','2026-10-31 15:10:49.100',0.000,0,0.00,'monthly','{\"monthly\": 100}');
/*!40000 ALTER TABLE `packages` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `payment_channels`
--

DROP TABLE IF EXISTS `payment_channels`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `payment_channels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(32) NOT NULL,
  `name` varchar(64) NOT NULL,
  `enabled` tinyint(1) NOT NULL DEFAULT '0',
  `builtin` tinyint(1) NOT NULL DEFAULT '0',
  `config` json DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_payment_channels_code` (`code`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `payment_channels`
--

LOCK TABLES `payment_channels` WRITE;
/*!40000 ALTER TABLE `payment_channels` DISABLE KEYS */;
INSERT INTO `payment_channels` VALUES (1,'epay','易支付',1,1,'{\"key\": \"6eOxa7fMYY6mqg6iqXM6MEI6M5eFDaVD\", \"pid\": \"1000\", \"alipay\": \"1\", \"wechat\": \"1\", \"api_url\": \"https://update.ywenlong.xyz/\", \"recharge_presets\": \"\"}','2026-10-31 08:39:11.177','2026-10-31 11:31:24.960');
/*!40000 ALTER TABLE `payment_channels` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `payment_orders`
--

DROP TABLE IF EXISTS `payment_orders`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `payment_orders` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `out_trade_no` varchar(64) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `package_id` bigint unsigned NOT NULL,
  `period` varchar(16) NOT NULL,
  `amount` decimal(14,2) NOT NULL DEFAULT '0.00',
  `pay_type` varchar(16) NOT NULL,
  `status` varchar(16) NOT NULL DEFAULT 'pending',
  `trade_no` varchar(64) NOT NULL DEFAULT '',
  `paid_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `kind` varchar(16) NOT NULL DEFAULT 'package',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_payment_orders_out_trade_no` (`out_trade_no`),
  KEY `idx_payment_orders_user_id` (`user_id`),
  KEY `idx_payment_orders_package_id` (`package_id`),
  KEY `idx_payment_orders_status` (`status`),
  KEY `idx_payment_orders_kind` (`kind`)
) ENGINE=InnoDB AUTO_INCREMENT=15 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `payment_orders`
--

LOCK TABLES `payment_orders` WRITE;
/*!40000 ALTER TABLE `payment_orders` DISABLE KEYS */;
INSERT INTO `payment_orders` VALUES (3,'AO198b22380dd345478bfc7054',1,0,'',1.00,'wxpay','failed','',NULL,'2026-10-31 11:28:14.341','2026-10-31 11:28:14.619','recharge'),(4,'AO8b14a35eafee4313a16e3ef7',1,0,'',1.00,'wxpay','failed','',NULL,'2026-10-31 11:30:06.337','2026-10-31 11:30:06.623','recharge'),(5,'AO2c5a67d8ec994fb8a7abd8b1',1,0,'',1.00,'wxpay','failed','',NULL,'2026-10-31 11:30:23.544','2026-10-31 11:30:24.034','recharge'),(6,'AO634e4567317147659804e8da',1,0,'',1.00,'wxpay','failed','',NULL,'2026-10-31 11:30:45.315','2026-10-31 11:30:45.529','recharge'),(8,'AOce3e86e0b7d64ef1ae295fd1',1,0,'',10.00,'wxpay','pending','',NULL,'2026-10-31 11:31:48.834','2026-10-31 11:31:48.834','recharge'),(9,'AO76b20788bdf543139dfeae4d',1,0,'',1.00,'wxpay','pending','',NULL,'2026-10-31 11:31:57.442','2026-10-31 11:31:57.442','recharge'),(10,'AO875361875dea482faf23a2f5',1,0,'',1.00,'wxpay','pending','',NULL,'2026-10-31 11:34:17.944','2026-10-31 11:34:17.944','recharge'),(11,'AO4d659915c8064140ae65a3f8',1,0,'',1.00,'wxpay','paid','2026103111381361610','2026-10-31 11:43:09.678','2026-10-31 11:38:13.188','2026-10-31 11:43:09.678','recharge'),(12,'AOdb45bcbdbeb34db6a7ca004d',1,0,'',1.00,'wxpay','pending','',NULL,'2026-10-31 11:38:54.235','2026-10-31 11:38:54.235','recharge'),(13,'AO9acb261b2c0e4ace88399de5',1,0,'',1.00,'wxpay','paid','2026103111450790904','2026-10-31 11:45:22.197','2026-10-31 11:45:07.321','2026-10-31 11:45:22.197','recharge'),(14,'AO1942452010cc4efb913d5713',1,0,'',1.00,'wxpay','paid','2026103115045671412','2026-10-31 15:05:12.267','2026-10-31 15:04:56.659','2026-10-31 15:05:12.267','recharge');
/*!40000 ALTER TABLE `payment_orders` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `permissions`
--

DROP TABLE IF EXISTS `permissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `permissions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `name` varchar(64) NOT NULL,
  `group` varchar(32) NOT NULL,
  `sort` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_permissions_code` (`code`),
  KEY `idx_permissions_group` (`group`)
) ENGINE=InnoDB AUTO_INCREMENT=39 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `permissions`
--

LOCK TABLES `permissions` WRITE;
/*!40000 ALTER TABLE `permissions` DISABLE KEYS */;
INSERT INTO `permissions` VALUES (1,'dashboard:view','查看概览','dashboard',1),(2,'node:read','查看节点','node',1),(3,'node:create','创建节点','node',2),(4,'node:update','编辑节点','node',3),(5,'port:list','列出端口','port',1),(6,'port:read','查看端口','port',2),(7,'port:create','创建端口','port',3),(8,'port:update','编辑端口','port',4),(9,'port:delete','删除端口','port',5),(10,'port:stats','查看端口流量','port',6),(11,'port:manage-all','管理全部端口（跨用户）','port',7),(12,'user:list','查看用户','user',1),(13,'user:create','创建用户','user',2),(14,'user:update','编辑用户','user',3),(15,'user:delete','删除用户','user',4),(16,'role:list','查看角色','role',1),(17,'role:create','创建角色','role',2),(18,'role:update','编辑角色权限','role',3),(19,'role:delete','删除角色','role',4),(20,'apikey:list','查看 API 密钥','apikey',1),(21,'apikey:create','创建 API 密钥','apikey',2),(22,'apikey:revoke','停用/启用 API 密钥','apikey',3),(23,'apikey:delete','删除 API 密钥','apikey',4),(24,'apikey:manage-all','管理全部 API 密钥','apikey',5),(25,'session:list','查看会话','session',1),(26,'session:kick','踢下线','session',2),(27,'session:manage-all','管理全部会话','session',3),(28,'profile:view','查看个人设置','profile',1),(29,'profile:update','修改个人设置','profile',2),(30,'system:view','查看系统设置','system',1),(31,'node:report','节点上报（Agent 专用）','node',5),(32,'package:list','查看套餐','package',1),(33,'package:create','创建套餐','package',2),(34,'package:update','编辑套餐','package',3),(35,'package:delete','删除套餐','package',4),(36,'node:delete','删除节点','node',4),(37,'system:update','修改系统设置','system',2),(38,'wallet:view','钱包商城','wallet',1);
/*!40000 ALTER TABLE `permissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `port_rules`
--

DROP TABLE IF EXISTS `port_rules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `port_rules` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `protocol` varchar(8) NOT NULL DEFAULT 'tcp',
  `listen_port` bigint NOT NULL DEFAULT '0',
  `target_addr` varchar(255) NOT NULL DEFAULT '',
  `ratio_up` decimal(6,3) NOT NULL DEFAULT '1.000',
  `ratio_down` decimal(6,3) NOT NULL DEFAULT '1.000',
  `rate_up_bps` bigint NOT NULL DEFAULT '0',
  `rate_down_bps` bigint NOT NULL DEFAULT '0',
  `quota_bytes` bigint NOT NULL DEFAULT '0',
  `used_bytes` bigint NOT NULL DEFAULT '0',
  `enabled` tinyint(1) NOT NULL DEFAULT '1',
  `status` varchar(32) NOT NULL DEFAULT 'active',
  `revision` bigint NOT NULL DEFAULT '1',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `owner_user_id` bigint unsigned DEFAULT NULL,
  `reverse_group` varchar(64) NOT NULL DEFAULT '',
  `direction` varchar(16) NOT NULL DEFAULT 'entry',
  `package_id` bigint unsigned NOT NULL DEFAULT '0',
  `node` varchar(64) NOT NULL,
  `egress_node` varchar(64) NOT NULL DEFAULT '',
  `transport` varchar(8) NOT NULL DEFAULT 'quic',
  PRIMARY KEY (`id`),
  KEY `idx_port_rules_owner_user_id` (`owner_user_id`),
  KEY `idx_port_rules_reverse_group` (`reverse_group`),
  KEY `idx_port_rules_direction` (`direction`),
  KEY `idx_port_rules_package_id` (`package_id`),
  KEY `idx_port_rules_node` (`node`),
  KEY `idx_port_rules_egress_node` (`egress_node`)
) ENGINE=InnoDB AUTO_INCREMENT=33 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `port_rules`
--

LOCK TABLES `port_rules` WRITE;
/*!40000 ALTER TABLE `port_rules` DISABLE KEYS */;
INSERT INTO `port_rules` VALUES (32,'333','both',58355,'adsdasd.com:9393',1.000,1.000,0,0,0,0,1,'active',1,'2026-10-31 15:01:45.650','2026-10-31 15:01:45.650',1,'','entry',12,'n-5dfb85fe','n-497f6d3e','quic');
/*!40000 ALTER TABLE `port_rules` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roles`
--

DROP TABLE IF EXISTS `roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `roles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `code` varchar(64) NOT NULL,
  `name` varchar(64) NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `builtin` tinyint(1) NOT NULL DEFAULT '0',
  `permission_codes` json DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_roles_code` (`code`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roles`
--

LOCK TABLES `roles` WRITE;
/*!40000 ALTER TABLE `roles` DISABLE KEYS */;
INSERT INTO `roles` VALUES (1,'super_admin','超级管理员','拥有全部权限，内置角色不可删除或修改',1,'[\"dashboard:view\", \"node:read\", \"node:create\", \"node:update\", \"node:delete\", \"node:report\", \"package:list\", \"package:create\", \"package:update\", \"package:delete\", \"port:list\", \"port:read\", \"port:create\", \"port:update\", \"port:delete\", \"port:stats\", \"port:manage-all\", \"user:list\", \"user:create\", \"user:update\", \"user:delete\", \"role:list\", \"role:create\", \"role:update\", \"role:delete\", \"session:list\", \"session:kick\", \"session:manage-all\", \"profile:view\", \"profile:update\", \"wallet:view\", \"system:view\", \"system:update\"]','2026-10-30 22:23:45.576','2026-10-31 15:32:06.708'),(2,'admin','管理员','管理节点、端口与用户，但不能创建用户或修改角色权限',1,'[\"dashboard:view\", \"node:read\", \"node:create\", \"node:update\", \"node:delete\", \"package:list\", \"package:create\", \"package:update\", \"package:delete\", \"port:list\", \"port:read\", \"port:create\", \"port:update\", \"port:delete\", \"port:stats\", \"port:manage-all\", \"user:list\", \"user:update\", \"role:list\", \"session:list\", \"session:kick\", \"session:manage-all\", \"profile:view\", \"profile:update\", \"wallet:view\", \"system:view\"]','2026-10-30 22:23:45.579','2026-10-31 15:32:06.710'),(3,'user','普通用户','仅管理自己创建的端口，可查看节点与个人设置',1,'[\"dashboard:view\", \"node:read\", \"package:list\", \"port:list\", \"port:read\", \"port:create\", \"port:update\", \"port:delete\", \"port:stats\", \"profile:view\", \"profile:update\", \"wallet:view\"]','2026-10-30 22:23:45.581','2026-10-31 15:32:06.711'),(4,'agent','Agent 节点','内置 Agent 账号专用：只能查看节点目录并上报运行数据',1,'[\"node:read\", \"node:report\"]','2026-10-31 00:04:46.770','2026-10-31 15:32:06.712');
/*!40000 ALTER TABLE `roles` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sessions`
--

DROP TABLE IF EXISTS `sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `sid` varchar(64) NOT NULL,
  `user_id` bigint unsigned NOT NULL,
  `refresh_hash` varchar(64) NOT NULL,
  `ua` varchar(255) DEFAULT NULL,
  `ip` varchar(64) DEFAULT NULL,
  `expires_at` datetime(3) DEFAULT NULL,
  `revoked_at` datetime(3) DEFAULT NULL,
  `last_seen_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sessions_s_id` (`sid`),
  KEY `idx_sessions_user_id` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=254 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sessions`
--

LOCK TABLES `sessions` WRITE;
/*!40000 ALTER TABLE `sessions` DISABLE KEYS */;
INSERT INTO `sessions` VALUES (1,'03d9bde81a62a6272d36995d005f06ed',1,'ca9be1c5fcdbc4205b6f66df7abcb262955829ff3f5153dcb60515402e565976','Python-urllib/3.11','127.0.0.1','2026-11-06 22:40:30.198','2026-10-30 22:40:30.454','2026-10-30 22:40:30.198','2026-10-30 22:40:30.199'),(2,'76131421a92454361b79a82518fb35dd',2,'8aa301b8083f5bf92bfc3a3ff055ef0422ac26a41836c5acbfd48f8170e42c84','Python-urllib/3.11','127.0.0.1','2026-11-06 22:40:30.372','2026-10-30 22:40:30.451','2026-10-30 22:40:30.449','2026-10-30 22:40:30.372'),(3,'e2a3910e758955e35a390777b2b5923d',3,'493a2006ada074916af127257e6e79410b7c247b641bb04953722b696172e20b','Python-urllib/3.11','127.0.0.1','2026-11-06 22:40:30.435','2026-10-30 22:40:30.526','2026-10-30 22:40:30.435','2026-10-30 22:40:30.435'),(4,'c17da9cce12301480f8ab033beccc0d7',1,'5e1b38c14656a1dd7239f5d0ce2ef48e78186df5aa0a5b5cf0350663a162102c','Python-urllib/3.11','127.0.0.1','2026-11-06 22:40:30.513',NULL,'2026-10-30 22:40:30.513','2026-10-30 22:40:30.513'),(5,'4e88aa8b4011bae12c722003e59feee9',1,'4ecc8f94dc0f2fcda44c217c75f03df989507e4dc19f34d5ecc8ec44ef4d956e','Python-urllib/3.11','127.0.0.1','2026-11-06 22:41:35.056','2026-10-30 22:41:35.309','2026-10-30 22:41:35.056','2026-10-30 22:41:35.056'),(6,'e02a1c7815da1c617fbf986da2f81b2d',4,'c524dcb36ea12c95d6ab9353bb18e6a7447fb311a6a51753034eb9f6d3936a77','Python-urllib/3.11','127.0.0.1','2026-11-06 22:41:35.229','2026-10-30 22:41:35.307','2026-10-30 22:41:35.303','2026-10-30 22:41:35.229'),(7,'dd5f8305cb2c8f09e6f72c38ec11aef1',5,'ea3a9e6dd339a0548f4e543a03237c79cf2d7f56b77d4b95c5e84a1fafef4a92','Python-urllib/3.11','127.0.0.1','2026-11-06 22:41:35.292','2026-10-30 22:41:35.377','2026-10-30 22:41:35.292','2026-10-30 22:41:35.292'),(8,'06da96a2f533556859d08a9e99757ce6',1,'339d11256c20029a6c28bb7cfce840588ae37505f411ee0fe04c1cefd84eab31','Python-urllib/3.11','127.0.0.1','2026-11-06 22:41:35.365',NULL,'2026-10-30 22:41:35.365','2026-10-30 22:41:35.365'),(9,'286284a1a212af16a758a643de9bce13',1,'aa86559b13038e44b54df0b0c410584dcc178822a980793d7893615c27770162','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-06 22:42:56.743',NULL,'2026-10-30 22:42:56.743','2026-10-30 22:42:56.743'),(10,'836a5ab2019abec13175412fe282293d',1,'8d24feeba47f21789d5d074f1a86db73339be07cf07eb4efbae14f4aa97108e3','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-06 23:04:43.426',NULL,'2026-10-30 23:04:43.426','2026-10-30 23:04:43.426'),(11,'101afe02b902e9cc97c24cef2c75d49c',1,'647f25a892a4c9f191124fdaa14beeb1708ec67c6531057270e5ce8c7d99bbed','Python-urllib/3.11','127.0.0.1','2026-11-06 23:18:18.953',NULL,'2026-10-30 23:18:18.953','2026-10-30 23:18:18.954'),(12,'27699d5228453bde557297f0fbee0d7b',1,'8bb4e212253a3978d8bfde94427bb55bb7ec98e8b7a67a6fd21e1736e0b826c9','Python-urllib/3.11','127.0.0.1','2026-11-06 23:19:13.236',NULL,'2026-10-30 23:19:13.236','2026-10-30 23:19:13.236'),(13,'8f6e92eace7ae9a2980832db1015ab6e',1,'aeee3c3f73108f2042a1abaf4c1698fd551b8635b43484a8069e49aa2f33c55c','Python-urllib/3.11','127.0.0.1','2026-11-06 23:19:30.357',NULL,'2026-10-30 23:19:30.357','2026-10-30 23:19:30.357'),(14,'57dc5926b4609d057e34ee948cfdfd8c',1,'d675cb9e9174204f0dfd8024021581077ed740a29aeff411d1dbfce0b2a40889','Python-urllib/3.11','127.0.0.1','2026-11-06 23:20:17.244',NULL,'2026-10-30 23:20:17.244','2026-10-30 23:20:17.244'),(15,'8cbe6bb492da507d354236191c5135a3',1,'0bbf10f8fa741f5ebaa8e5f54fef480fff003390d557346de822c4debf3253f1','Python-urllib/3.11','127.0.0.1','2026-11-06 23:20:41.494',NULL,'2026-10-30 23:20:41.494','2026-10-30 23:20:41.494'),(16,'bf1cfe5a0a5e89eeb3955f79667a2a9a',1,'6e4c92ab1a4c62ca5ccbcbeb100377d2a3759eba3ca73161b7db074af36f6948','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-06 23:21:22.198',NULL,'2026-10-30 23:21:22.198','2026-10-30 23:21:22.198'),(17,'71e3bb7b84e97d11ceb803425366fb2e',1,'2ec5ee48e2a8054ca8a1923e87a930be08990082d3be3d76f5939486cb47a42e','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-06 23:40:53.503',NULL,'2026-10-30 23:40:53.503','2026-10-30 23:40:53.503'),(18,'50a472b7b9ea018df0acb88064676522',1,'4bd7018d78fa890c6d3d9bfbd0a0e40531364d0866447cd8f2077c4f68121f9b','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-06 23:56:09.183','2026-10-31 00:01:40.747','2026-10-30 23:56:09.183','2026-10-30 23:56:09.183'),(19,'50d4072a6e1e987bae6ed612b32127b9',1,'93b6c7501a97a2c6eb56cc5db91c9efb428b3dda33aba8f297ba122bdd9914e5','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 00:02:07.497',NULL,'2026-10-31 00:02:07.497','2026-10-31 00:02:07.497'),(20,'c490d55f1ff6d62913d2cdefd3a11366',1,'d9fd06dd8ce2a3b9d40845e01dcf9c5fc4a5b8150dbcfffd312f2d207f4ba954','Python-urllib/3.11','127.0.0.1','2026-11-07 00:05:10.503',NULL,'2026-10-31 00:05:10.503','2026-10-31 00:05:10.503'),(21,'f2e1a32477a4c484d8498377e886238c',6,'463d68d0dd2b6bbeb1e7b4fc37574bcb35a375ef7b11af25dc2de04c93b0751a','Python-urllib/3.11','127.0.0.1','2026-11-07 00:05:10.562',NULL,'2026-10-31 00:05:10.562','2026-10-31 00:05:10.562'),(22,'c73fafb40e580628c2f4dfffe8b133b0',1,'9c5616097f7ee3bb7beeb86cec08f077b88f95ad05793fbf479efcd1ad365406','Python-urllib/3.11','127.0.0.1','2026-11-07 00:05:21.298',NULL,'2026-10-31 00:05:21.298','2026-10-31 00:05:21.298'),(23,'ba146ca860bfacf673308b0e6b06dfd3',6,'aba1a0ad1dbbae562bb4e3ce7ccc46214b083c1306d9dc2040de54300cfa6a64','Go-http-client/1.1','127.0.0.1','2026-11-07 00:05:32.726',NULL,'2026-10-31 00:05:32.726','2026-10-31 00:05:32.726'),(24,'ab338102a4ea07a36e51173070f31e2d',6,'7378140bd7fc12b8be4556d65f95e05fd8e927f466cb964db67703571fcb24a7','Go-http-client/1.1','127.0.0.1','2026-11-07 00:05:34.592',NULL,'2026-10-31 00:05:34.592','2026-10-31 00:05:34.592'),(25,'53030a52b22ede638202eba84646b72f',1,'5ee927f36867657891ddbbdee37cf5a3a6027d69591b37aea574a8b9cbf4396a','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 00:05:57.449',NULL,'2026-10-31 00:22:42.577','2026-10-31 00:05:57.450'),(26,'9664e1dda37e4c68893f3d3167723098',1,'df086eb050e41902af96ab834e9c1e9d45f3e53d50dc9d28808d753a95f1d4c2','curl/7.88.1','127.0.0.1','2026-11-07 00:09:10.115',NULL,'2026-10-31 00:09:10.115','2026-10-31 00:09:10.115'),(27,'71ff5d2c01a3e5f110c8806ab0deceb2',1,'c6ddf092a5c6487c96cc538350f165d1fe87c1f90108c381bfe946aa0d898ad4','curl/7.88.1','127.0.0.1','2026-11-07 00:09:16.585',NULL,'2026-10-31 00:09:16.585','2026-10-31 00:09:16.585'),(28,'a1269081ab95dfa936d5024f51494993',6,'782db533bba9072a124075b396512b5d42c3c2d68b4b75ceb9bf31cab3112252','Go-http-client/1.1','127.0.0.1','2026-11-07 00:09:24.047',NULL,'2026-10-31 00:09:24.047','2026-10-31 00:09:24.047'),(29,'72adf2526fbb7eb762ccc2be2ef01be6',6,'e8d60bceba442e8a0d0590c170fcc2ec2722576dcd9e1214fc166329dc2e1ade','Go-http-client/1.1','127.0.0.1','2026-11-07 00:09:26.027',NULL,'2026-10-31 00:09:26.027','2026-10-31 00:09:26.027'),(30,'b99fff868fbe20494cb86c9192d17417',1,'e6fa36911d9c4582c538c081280e72c61f9613539b5cfdcafbfed75ee31a0018','curl/7.88.1','127.0.0.1','2026-11-07 00:09:42.693','2026-10-31 00:10:25.161','2026-10-31 00:09:42.693','2026-10-31 00:09:42.693'),(31,'0c1b8bc891cb2c6cf512d4f72da3e313',1,'2bae3d7c7c0d2540214c8ee24c116db211587560a00c10c9b271989798ce25cc','curl/7.88.1','127.0.0.1','2026-11-07 00:10:36.203',NULL,'2026-10-31 00:10:36.203','2026-10-31 00:10:36.203'),(32,'b2bebfb66c5b585c30d9023ec50167c1',6,'ee9d99320b77c2c8b326a0448ee19ff687f6b9b2932cc7537548fedd1accd7ce','curl/7.88.1','127.0.0.1','2026-11-07 00:11:19.817',NULL,'2026-10-31 00:11:19.817','2026-10-31 00:11:19.818'),(33,'bfe449d7b100be22c1a1f9b36688ad7f',6,'32931e65a841fb80662058ad033ee2665903b592c3fae7ba099d1a7eedc836b1','curl/7.88.1','127.0.0.1','2026-11-07 00:11:40.164',NULL,'2026-10-31 00:11:40.164','2026-10-31 00:11:40.164'),(34,'d15b029f6eb106c0fba861a937153f7e',6,'185a2569fb0365aea01632a7a531985cf8dc65af3dff710c819fbb83db75a4a4','Go-http-client/1.1','127.0.0.1','2026-11-07 00:11:56.726',NULL,'2026-10-31 00:11:56.726','2026-10-31 00:11:56.726'),(35,'81ed22d3de7908eaabf56c697a6b6922',6,'f95dca480211eabfb8c37440dd27e4f8a8bfd721860df5d36aec65107e9328ec','Go-http-client/1.1','127.0.0.1','2026-11-07 00:23:29.020',NULL,'2026-10-31 00:23:29.020','2026-10-31 00:23:29.020'),(36,'9a77c6aca873060866ee4f8274c738f5',6,'073b60bdba467023d30ac578788e5bacfc5ee542f840e3876bd4c9ed6ab30273','Go-http-client/1.1','127.0.0.1','2026-11-07 00:25:59.719',NULL,'2026-10-31 00:25:59.719','2026-10-31 00:25:59.719'),(37,'11329c87a2bff9c3eab66cbbdc2a850c',6,'bc4dcff8480b249e0b1de4b1523879c82c561aeb2cc4b23875d2495327c1a7cb','Go-http-client/1.1','127.0.0.1','2026-11-07 00:37:32.021',NULL,'2026-10-31 00:37:32.021','2026-10-31 00:37:32.021'),(38,'cc0fc81408d6fd47cd778a1d53c69894',6,'b0e5c3e5bf6f7fc7e01f06c416712923488a09d2b828f8aa910329fef9d66625','Go-http-client/1.1','127.0.0.1','2026-11-07 00:40:02.718',NULL,'2026-10-31 00:40:02.718','2026-10-31 00:40:02.718'),(39,'463b2e029870bce7977d9b762288df10',6,'d50167795db59be4a094cc464f1194a1023d0da04d3628f6ea4fa4d389630051','Go-http-client/1.1','127.0.0.1','2026-11-07 00:45:43.613',NULL,'2026-10-31 00:45:43.613','2026-10-31 00:45:43.613'),(40,'759c44e44bb4e3b2e317414ae596fedb',6,'b3baab0d21daee9bbd10c5a19ca238971a7dc60389b8a79acbe21e3a2376d427','Go-http-client/1.1','127.0.0.1','2026-11-07 00:45:49.189',NULL,'2026-10-31 00:45:49.189','2026-10-31 00:45:49.189'),(41,'bdc7721564a58d15d197e32f24eb3e48',1,'9af690b6328044911371ba1837e002140a9f188a07842cfd1f7a36f03a7354f1','curl/7.88.1','127.0.0.1','2026-11-07 00:46:29.985',NULL,'2026-10-31 00:46:29.985','2026-10-31 00:46:29.985'),(42,'25e454e932f0ba24c6d1fa5371b92ab3',6,'aee74dbc5eda5c1e32afef66216fe5403752fb63d432af58e80a435a920bb113','Go-http-client/1.1','127.0.0.1','2026-11-07 00:59:44.606',NULL,'2026-10-31 00:59:44.606','2026-10-31 00:59:44.606'),(43,'563393dd9a77ac42ceab56d6489a2236',6,'d6e77d90dec09b6bd01de73fdaf16af4eec1744759f3b0b54dda76b837819a5c','Go-http-client/1.1','127.0.0.1','2026-11-07 00:59:50.183',NULL,'2026-10-31 00:59:50.183','2026-10-31 00:59:50.183'),(44,'0e84f68b45be10a759a29f3069850f18',1,'49e02f3bf1a49b88da338cc14a4803cf15ec2180fb9557212299946fc44cfa4b','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 01:04:34.393',NULL,'2026-10-31 01:04:34.393','2026-10-31 01:04:34.393'),(45,'f34fa847849321b2edb62b743b8b32e9',6,'4ffa7cd9107ff0cb784d7a05ba9fca1403ebe2f9928c095161fc6a2f79c9e74f','Go-http-client/1.1','127.0.0.1','2026-11-07 01:11:32.422',NULL,'2026-10-31 01:11:32.422','2026-10-31 01:11:32.422'),(46,'67ee7bc17da1c6a7d0c369c8da28a038',6,'0224b6d2517251554040ca8839d6f5e39fedd2fb42cd2f3b01a6a4d02a97d585','Go-http-client/1.1','127.0.0.1','2026-11-07 01:11:35.007',NULL,'2026-10-31 01:11:35.007','2026-10-31 01:11:35.007'),(47,'9eadb824c4357a27f8c826387fe54e9f',1,'b0c6bf6a95a310fbdc787e7bbe9f7c91692e6922b4549be4ffa4def404665b6f','curl/7.88.1','127.0.0.1','2026-11-07 01:11:54.354',NULL,'2026-10-31 01:11:54.354','2026-10-31 01:11:54.354'),(48,'efbb52b2adacfc78429664330acfce40',6,'00a681e36f82e774299e2056ceda86e5f2520b81132f88d47432fc8f3a5bc00a','Go-http-client/1.1','127.0.0.1','2026-11-07 01:13:29.021',NULL,'2026-10-31 01:13:29.021','2026-10-31 01:13:29.021'),(49,'f76f2964078fd3d9edef15f31878619c',6,'e840d46f0844257b4248d6c9452767c86e16eac023c800cfce2731ca136b5bd4','Go-http-client/1.1','127.0.0.1','2026-11-07 01:13:31.155',NULL,'2026-10-31 01:13:31.155','2026-10-31 01:13:31.155'),(50,'b14d995236b15f63f68b26d95e6fab71',1,'1ea8d275734bb40a4c4e256692eb802e79fe31059175496c6355799704b141fc','curl/7.88.1','127.0.0.1','2026-11-07 01:13:41.320',NULL,'2026-10-31 01:13:41.320','2026-10-31 01:13:41.320'),(51,'9a7ab766aa69ccb8e98595fb474d25f6',6,'8a4c978c88be8788657c377a06423f3a564463c927fa6fba0b947e51cc5a8e4a','Go-http-client/1.1','127.0.0.1','2026-11-07 01:18:44.851',NULL,'2026-10-31 01:18:44.851','2026-10-31 01:18:44.851'),(52,'cd9c6fd8065939d085dc9bf2862b680f',6,'5c2d7dbfbc78e56972c99e55ab931e7c4f3e6cbbb082d67d308637e516528a97','Go-http-client/1.1','127.0.0.1','2026-11-07 01:18:47.098',NULL,'2026-10-31 01:18:47.098','2026-10-31 01:18:47.098'),(53,'e547164f6b39cacf6035b6d56fa816c3',1,'116cb596c9c530e1703e8f748381a8e5cbe9fd6437601829266a1b7ce384db27','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 01:22:36.863',NULL,'2026-10-31 01:38:30.880','2026-10-31 01:22:36.863'),(54,'1f8f262022292d887f8358868e770c83',6,'c493ed16356eb0697785b27ad425861fc2c55dbfdc8b26cdc5ba97cfa47123f1','Go-http-client/1.1','127.0.0.1','2026-11-07 01:23:57.567',NULL,'2026-10-31 01:23:57.567','2026-10-31 01:23:57.567'),(55,'3dd379806323dc638c145d79613a429d',6,'d325395b0e17d145c38fa55ad4aa1ce2274e9effd31fea31f382850c4cfaa63b','Go-http-client/1.1','127.0.0.1','2026-11-07 01:23:59.700',NULL,'2026-10-31 01:23:59.700','2026-10-31 01:23:59.701'),(56,'3ea11b678376c95a9b863e4f4fc3346b',6,'1f3870c2690c1091b9313a5682173fc9143d8d9336f8d1f6d7a1ca1aee5a8848','Go-http-client/1.1','127.0.0.1','2026-11-07 01:37:58.561',NULL,'2026-10-31 01:37:58.561','2026-10-31 01:37:58.561'),(57,'2e0fc6e5b3eb485a07b60cea8baaa699',6,'0437ac98f0dc46bc2435f0d02f29b5505b4f168379da2a42e88e06be81fa37af','Go-http-client/1.1','127.0.0.1','2026-11-07 01:38:00.697',NULL,'2026-10-31 01:38:00.697','2026-10-31 01:38:00.697'),(58,'e4498a67337b4d331caf0bb967489234',6,'c2a7934d812164840d793e31e355e900f3ad488a9dedc50bb34311ef7ed63969','Go-http-client/1.1','127.0.0.1','2026-11-07 01:40:52.469',NULL,'2026-10-31 01:40:52.469','2026-10-31 01:40:52.469'),(59,'fa99702725ffa29d151e3dd6b94c109d',6,'5195bc5143cb6bd6f18b1dbc11dc30121e52c89dbdc073f85cbee61810466eb4','Go-http-client/1.1','127.0.0.1','2026-11-07 01:40:57.084',NULL,'2026-10-31 01:40:57.084','2026-10-31 01:40:57.084'),(60,'efeccfffdd589fa3bc32100affa78894',1,'21114da0c8f9d158b8cf5faeb055e41e5426af6cc4e4d807516600ac25936116','curl/7.88.1','127.0.0.1','2026-11-07 01:41:10.667',NULL,'2026-10-31 01:41:10.667','2026-10-31 01:41:10.668'),(61,'3c37d68d120058fb2396c728bd091e60',1,'493771490f760af0f8f5cb1997c977c51bda3852fb044ef9d4e6f3bc2d452176','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) TraeCN/1.107.1 Chrome/142.0.7444.235 Electron/39.2.7 Safari/537.36','127.0.0.1','2026-11-07 01:44:51.323','2026-10-31 01:54:19.560','2026-10-31 01:44:51.323','2026-10-31 01:44:51.323'),(62,'b7c9a923c6064b74b155b656f435c3b5',1,'ee4a524b2e5a591cf96d27aba498118d13c8e6cb7960a3d279ddf155d0cdfb04','curl/7.88.1','127.0.0.1','2026-11-07 01:53:29.908',NULL,'2026-10-31 01:53:29.908','2026-10-31 01:53:29.908'),(63,'a24a2c378ab60f0eb22ca30f5fd2fb69',7,'f4c07284522671f1f0c1db51aac319e3c549aeff2d8cb0c219aca244f74b9957','curl/7.88.1','127.0.0.1','2026-11-07 01:53:49.073','2026-10-31 02:01:54.620','2026-10-31 01:53:49.073','2026-10-31 01:53:49.073'),(64,'b5060a4e895a29055c654f302fe520ef',8,'bc83e1a2829ff4e59da077f9304dff27d1326c4d1c78ef4e5efdb8e9154534fb','curl/7.88.1','127.0.0.1','2026-11-07 01:53:54.657','2026-10-31 02:01:54.641','2026-10-31 01:53:54.657','2026-10-31 01:53:54.658'),(65,'0eab077cb9fb4d48aacd8746b54ea39e',6,'0460ced871af4314495fd58fa29c52bdfe8a0c8c95005f7768379ff07f96f812','Go-http-client/1.1','127.0.0.1','2026-11-07 01:54:53.462',NULL,'2026-10-31 01:54:53.462','2026-10-31 01:54:53.462'),(66,'e559f7aff125daa01b0d69fc3aae8a72',6,'b533e04a47bb728e890a900c39fbef0c628ba23818fa47eef9cb8dce1f8d1b3b','Go-http-client/1.1','127.0.0.1','2026-11-07 01:54:58.077',NULL,'2026-10-31 01:54:58.077','2026-10-31 01:54:58.077'),(67,'92d59949023212efa3fc8f48a153d7e5',6,'804d9943b82ddf94c0bb71538daab7dd61001d562b27abb48b4a390fc5984922','Go-http-client/1.1','127.0.0.1','2026-11-07 02:08:54.462',NULL,'2026-10-31 02:08:54.462','2026-10-31 02:08:54.462'),(68,'11b4ffe986fae8b0fdb7471a6aad1c4c',6,'c4dfdfa5e5ab279242fee5a319ba389085f466cb0bcc36c5278de292138be10a','Go-http-client/1.1','127.0.0.1','2026-11-07 02:08:59.076',NULL,'2026-10-31 02:08:59.076','2026-10-31 02:08:59.076'),(69,'c85435e8afa4adc79b9e4a02b5a0091a',6,'7417f5b83c5a65c259c241e49c7b1d3fc4f14252afdbc836a6aa7fbfbc69c656','Go-http-client/1.1','127.0.0.1','2026-11-07 02:22:55.462',NULL,'2026-10-31 02:22:55.462','2026-10-31 02:22:55.462'),(70,'bde89565c241aadd32c96b2cb60f58b1',6,'dd51eaf1e2bb558dfb4f12e8bae4940e29e0c3f5448bcb0adba3c18bc86a16e9','Go-http-client/1.1','127.0.0.1','2026-11-07 02:23:00.076',NULL,'2026-10-31 02:23:00.076','2026-10-31 02:23:00.076'),(71,'43577bb8095b23cc288b1bcc1ef4b592',6,'e3177d0924388fdec91dfce147a7d6329f8fb59317b2d4836e6fd759005c653b','Go-http-client/1.1','127.0.0.1','2026-11-07 02:36:56.462',NULL,'2026-10-31 02:36:56.462','2026-10-31 02:36:56.463'),(72,'ad7baa932bffa873a9f6bb4ef927656d',6,'3e56dcb59f9c76081b3f3fd7d43a2fe7572058ada362bd11bf33ddbbf23b2496','Go-http-client/1.1','127.0.0.1','2026-11-07 02:37:01.077',NULL,'2026-10-31 02:37:01.077','2026-10-31 02:37:01.077'),(73,'e371a93fc2913fe26acd299df9107a44',6,'80bfbb73287cc52424235daa6f2d0e776ac24396532ccf9db54c66770a8d48d3','Go-http-client/1.1','127.0.0.1','2026-11-07 02:50:57.462',NULL,'2026-10-31 02:50:57.462','2026-10-31 02:50:57.462'),(74,'3fd07fe293196ec98bc0fb2b1f324ab4',6,'ebfa3ffb7df410cb85f11eea93b724e1399066f3e835a841960b090df5c287f6','Go-http-client/1.1','127.0.0.1','2026-11-07 02:51:02.077',NULL,'2026-10-31 02:51:02.077','2026-10-31 02:51:02.077'),(75,'1a705fefb968c7f7249d843fa7318ef3',6,'f9e4f5b45f1029ca34895b2490aea865ac3bbc8710b2510de021d0809bfd7218','Go-http-client/1.1','127.0.0.1','2026-11-07 03:04:58.462',NULL,'2026-10-31 03:04:58.462','2026-10-31 03:04:58.462'),(76,'319b1285d94a7510993cda0c7b80ab35',6,'d57803180ed610e0f97cec73650ccc9ce0eee14b1b8fad8f1d5a7b789a2e227b','Go-http-client/1.1','127.0.0.1','2026-11-07 03:05:03.077',NULL,'2026-10-31 03:05:03.077','2026-10-31 03:05:03.077'),(77,'968f8865273ea00e44e42b704c3a41ed',6,'098068cb68e555de135e23bfacb2dedfde07340896b8fff112d1d2c39f6f9c73','Go-http-client/1.1','127.0.0.1','2026-11-07 03:18:59.461',NULL,'2026-10-31 03:18:59.461','2026-10-31 03:18:59.462'),(78,'9b4c78e2303f3f915e894626fc567aa8',6,'b02981bf2d1ac58045a00434eff62cc7818d8bd1d1aea165b917acbefc0e0eeb','Go-http-client/1.1','127.0.0.1','2026-11-07 03:19:04.077',NULL,'2026-10-31 03:19:04.077','2026-10-31 03:19:04.077'),(79,'92de4c72b0645ca95547f1517068d27f',6,'cbf45390601cc46e01b0e9afc32193c8ac9029181336dbae58a5623f2deab5c6','Go-http-client/1.1','127.0.0.1','2026-11-07 03:33:00.462',NULL,'2026-10-31 03:33:00.462','2026-10-31 03:33:00.462'),(80,'30d7b5d13a7237838be947a67f5680c8',6,'c73dd579b97db8f58da534411761480e680ab22291b1d0f668b05101fc5a2b90','Go-http-client/1.1','127.0.0.1','2026-11-07 03:33:05.076',NULL,'2026-10-31 03:33:05.076','2026-10-31 03:33:05.077'),(81,'6eacdd2a85877100197ee28010a17c9f',6,'0578dd0b51b58d4149f6081c6244b0cfc47a9f1ff9396305e0e73e5b55358ed2','Go-http-client/1.1','127.0.0.1','2026-11-07 03:47:01.462',NULL,'2026-10-31 03:47:01.462','2026-10-31 03:47:01.462'),(82,'b26feae9b44026a139e5a05bd10f7472',6,'bcf49703e786e410388cde0bcd4cd4ed979f89f0538c167141076abe612cef4f','Go-http-client/1.1','127.0.0.1','2026-11-07 03:47:06.076',NULL,'2026-10-31 03:47:06.076','2026-10-31 03:47:06.076'),(83,'c446325a9ace515ca9932f285b484164',6,'80298c2e5ec1b150798c61da3e4d495f280c074c5adf6ec2cb27ac1f7742b27c','Go-http-client/1.1','127.0.0.1','2026-11-07 04:01:02.463',NULL,'2026-10-31 04:01:02.463','2026-10-31 04:01:02.463'),(84,'1341af403f9c8fa4fbc722e075bf24fa',6,'4b9784aa1149e6a0a078018f1a1712e4dd4f0bbcdc0cf15d0dbeb60dc70e72bb','Go-http-client/1.1','127.0.0.1','2026-11-07 04:01:07.077',NULL,'2026-10-31 04:01:07.077','2026-10-31 04:01:07.077'),(85,'60b499de808a4dbb5a79262bdd5e6c29',6,'0c020d5e9ec8bd3822ab1ef982e4792380456cf47976a2206bbfb4949fc0e4bb','Go-http-client/1.1','127.0.0.1','2026-11-07 04:15:03.462',NULL,'2026-10-31 04:15:03.462','2026-10-31 04:15:03.462'),(86,'778112e5aa11e7b3249b64a3919aa033',6,'f183b2981095513bca512acee4b3307510eb71584c6217d966e90f8d805e5619','Go-http-client/1.1','127.0.0.1','2026-11-07 04:15:08.078',NULL,'2026-10-31 04:15:08.078','2026-10-31 04:15:08.078'),(87,'0c24a1a797029449b65c51dc76d3e761',6,'868c398266d6df283cf9bd5e117d93edad6a2292ad0ea39b7c93bd5bfc5f73e8','Go-http-client/1.1','127.0.0.1','2026-11-07 04:29:04.464',NULL,'2026-10-31 04:29:04.464','2026-10-31 04:29:04.464'),(88,'34ae3777c6d1f4c61c3bac55ebc5dca8',6,'f5a547e0de70c90e6b05b820bad763ad8258f1f9d08132c3f0caf6bec9c96044','Go-http-client/1.1','127.0.0.1','2026-11-07 04:29:09.076',NULL,'2026-10-31 04:29:09.076','2026-10-31 04:29:09.076'),(89,'94e47b1fdd9747c8364b161a7c0fd99b',6,'5c404ec5565cf7d3693f1cd0a9fba53b1e86643b181769846ffd7fddf20c40ad','Go-http-client/1.1','127.0.0.1','2026-11-07 04:43:05.462',NULL,'2026-10-31 04:43:05.462','2026-10-31 04:43:05.462'),(90,'fb7b5d532c6b3da8228e2e99415befe2',6,'c2f4bd98d4ceb3950be3ab32b8bc4b377bb75aba9eec1591933a21cc7b4ab34a','Go-http-client/1.1','127.0.0.1','2026-11-07 04:43:10.076',NULL,'2026-10-31 04:43:10.076','2026-10-31 04:43:10.077'),(91,'417e7e71b60c9a3bf2472e79735c4b39',6,'b8ff9a74c19ddeb852a7b0dd159dab98a616306681d52abdc79443aca84d68ce','Go-http-client/1.1','127.0.0.1','2026-11-07 04:57:06.463',NULL,'2026-10-31 04:57:06.463','2026-10-31 04:57:06.463'),(92,'dfdf7a34b7800d6865cd49db010c401e',6,'2be4754ed28f01ce5e70a2d4c45c0f029f65ae377729f2b4f0db485d1837a329','Go-http-client/1.1','127.0.0.1','2026-11-07 04:57:11.077',NULL,'2026-10-31 04:57:11.077','2026-10-31 04:57:11.077'),(93,'148b57304420337f44e8eee402597980',6,'b1ce897d2ebb7fd9d7ee268b23a1f15b0e512b2edf0b845f7b16916e6e8e2092','Go-http-client/1.1','127.0.0.1','2026-11-07 05:11:07.462',NULL,'2026-10-31 05:11:07.462','2026-10-31 05:11:07.462'),(94,'f2d0fb486f6981c9a4c03f5b84106598',6,'c90f955362bafad046adf82850728cc159dfc8e6b1a278db3420e93e6911b548','Go-http-client/1.1','127.0.0.1','2026-11-07 05:11:07.471',NULL,'2026-10-31 05:11:07.471','2026-10-31 05:11:07.471'),(95,'ff384169b0178dfde1bcb0ec116e3c05',6,'1559c41d4d7bf3e3ff7106d14e4b4dc4058bfb4a55b64cb8b2d6fa0fc95276c1','Go-http-client/1.1','127.0.0.1','2026-11-07 05:11:12.078',NULL,'2026-10-31 05:11:12.078','2026-10-31 05:11:12.078'),(96,'9c729ab5383259ba61828c1f4cdc6f97',6,'ed90d64e28c487026115c8e377fe3b942ce865cdfdcd5e19377bc4d01d552be7','Go-http-client/1.1','127.0.0.1','2026-11-07 05:11:12.089',NULL,'2026-10-31 05:11:12.089','2026-10-31 05:11:12.089'),(97,'6802ea1976985f07458d130c9949e9f3',6,'758965e0092c723e3dd499f1df68e1e5f7024eb75c993454abf53e1e03b371b6','Go-http-client/1.1','127.0.0.1','2026-11-07 05:25:08.462',NULL,'2026-10-31 05:25:08.462','2026-10-31 05:25:08.462'),(98,'601fc3c4bc4fe21c38f73d914c18e185',6,'89b177b787bd400659c6bdee31477ef7e3093a5d3c9d029f06e3fd9ecce64cd6','Go-http-client/1.1','127.0.0.1','2026-11-07 05:25:13.079',NULL,'2026-10-31 05:25:13.079','2026-10-31 05:25:13.079'),(99,'9c4f878a2f3c77d7a12628a31e8e4f57',6,'39d54a393ada977966cf016a40dba63a7380c1bcdf638d19ad7dfaca9d36141e','Go-http-client/1.1','127.0.0.1','2026-11-07 05:39:09.462',NULL,'2026-10-31 05:39:09.462','2026-10-31 05:39:09.462'),(100,'f8ff0f5a8ca181192d8496b9a363aeab',6,'c2f9c30f762d23b5db14722d01afa1f8ec10d8951c4c2703588b45e813c8920f','Go-http-client/1.1','127.0.0.1','2026-11-07 05:39:14.077',NULL,'2026-10-31 05:39:14.077','2026-10-31 05:39:14.077'),(101,'11a5535d370a88fb53a2066b5cf5c14e',6,'2ac73c382d009073bdce69c590832ad8d39e65a5455987bb523663514d3db00b','Go-http-client/1.1','127.0.0.1','2026-11-07 05:53:10.462',NULL,'2026-10-31 05:53:10.462','2026-10-31 05:53:10.462'),(102,'10cb663c2511f3f2e7f423b54e0b4395',6,'8fa9729569a0017094dff01c9e2225ad558ad7635d6f4213d3363eb977be5680','Go-http-client/1.1','127.0.0.1','2026-11-07 05:53:15.076',NULL,'2026-10-31 05:53:15.076','2026-10-31 05:53:15.076'),(103,'f2cf3d490b56bf852910cd43b3f81c30',6,'85be79b144ca77a30dc8bfd6a30530279ac990ea3b719f780dadfd0e3a434bd8','Go-http-client/1.1','127.0.0.1','2026-11-07 06:07:11.462',NULL,'2026-10-31 06:07:11.462','2026-10-31 06:07:11.462'),(104,'0b9a0c8ce3f0152e44d2b77c91194c06',6,'f38eaba94d11b90917d20b0f1db52c5d10bf83d1d1f71e74fe6e458a72e1d39d','Go-http-client/1.1','127.0.0.1','2026-11-07 06:07:16.076',NULL,'2026-10-31 06:07:16.076','2026-10-31 06:07:16.076'),(105,'6349ef96dc8f9f935c186bd9d9f26e42',6,'e61596def03746ef3223e9bfdad81b5b6b7494d3518fe37c90fd62789f80d1f2','Go-http-client/1.1','127.0.0.1','2026-11-07 06:21:12.461',NULL,'2026-10-31 06:21:12.461','2026-10-31 06:21:12.461'),(106,'5601316e2d610a1e659dd573ad05baa6',6,'97075b406e779b72117d96d99b4a6bb98b6a44b1aab863164fd166697f5292c0','Go-http-client/1.1','127.0.0.1','2026-11-07 06:21:17.077',NULL,'2026-10-31 06:21:17.077','2026-10-31 06:21:17.077'),(107,'e635fc2eef56b2342e5a3cb045160d61',6,'ee447d82fa6c79d2578d1c4baff0772b924b16101d9073fc3e371197b064d480','Go-http-client/1.1','127.0.0.1','2026-11-07 06:35:13.462',NULL,'2026-10-31 06:35:13.462','2026-10-31 06:35:13.462'),(108,'214a4a7825813630a0b3bc5a13cff883',6,'5fcf051c79f46c16cc6ad4fa5f9accab8c09f36870010d0a2cffb3aa46832113','Go-http-client/1.1','127.0.0.1','2026-11-07 06:35:18.076',NULL,'2026-10-31 06:35:18.076','2026-10-31 06:35:18.076'),(109,'1c56c051175b4330e64c055c63d8694c',6,'7af9638660a1822590958d700b2c27a6121715170f75f8ea7e1d85499c80bf27','Go-http-client/1.1','127.0.0.1','2026-11-07 06:49:14.461',NULL,'2026-10-31 06:49:14.461','2026-10-31 06:49:14.462'),(110,'fb4d5aa800f2a094939f31a22298b52a',6,'14be77bd1a7be030ea70e38e3c41970e5c3369d2a7cb31d136558fae055de393','Go-http-client/1.1','127.0.0.1','2026-11-07 06:49:19.076',NULL,'2026-10-31 06:49:19.076','2026-10-31 06:49:19.076'),(111,'d88b543523b12fa898d41db30ffd2359',6,'6af954299aa46b691420ec06ca2294e524cc4f42bfee03fe4a0fd77516789922','Go-http-client/1.1','127.0.0.1','2026-11-07 07:03:15.462',NULL,'2026-10-31 07:03:15.462','2026-10-31 07:03:15.462'),(112,'a54da6b3bc75cd429b5a2692b6ff47b2',6,'478da75e45b8fab2078f8c53680211cacb621f81eef6f82875a4528a38182856','Go-http-client/1.1','127.0.0.1','2026-11-07 07:03:20.076',NULL,'2026-10-31 07:03:20.076','2026-10-31 07:03:20.076'),(113,'6952a9d3057cf5a6cb6acb79eed33a6e',6,'d12db4d76cd398bca8e906e811340e768e9b5cb5e5d59d4d9db3d0f3a3cf8f27','Go-http-client/1.1','127.0.0.1','2026-11-07 07:17:16.461',NULL,'2026-10-31 07:17:16.461','2026-10-31 07:17:16.462'),(114,'68a03667c9c08fa69879ceefdb13105c',6,'0620c6f91dc973d77f00135071181e2b63bfd032608787e8d66366920ea669ff','Go-http-client/1.1','127.0.0.1','2026-11-07 07:17:21.076',NULL,'2026-10-31 07:17:21.076','2026-10-31 07:17:21.076'),(115,'22b6eb32751f4ecf2b184a41ae3df6c3',1,'a3d6141cbb680bd54de72172e50697b9ba82fb1941a18537315448aaf2285b17','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 07:27:55.452',NULL,'2026-10-31 07:27:55.452','2026-10-31 07:27:55.452'),(116,'7d15ce3a65a3a609133154b91b1d0343',6,'46e4d429eaab4555d3c6a32b4d8073e9eaf72e08a241a10fc94a88df04206607','Go-http-client/1.1','127.0.0.1','2026-11-07 07:31:17.462',NULL,'2026-10-31 07:31:17.462','2026-10-31 07:31:17.462'),(117,'b59135548e240a0a4ceb50f472bed1c4',6,'4ac3b9f05c0d2e775e999f6352f0b390c2ee5403253c4f4ab847d3a62eae9a87','Go-http-client/1.1','127.0.0.1','2026-11-07 07:31:22.076',NULL,'2026-10-31 07:31:22.076','2026-10-31 07:31:22.076'),(118,'9f3184123ff80d90a64b26cd7ff39d90',1,'58af5c0bf77f48cbb8aeb8b589d748a312930eb8a4dd231d3c531f9da0ad3003','curl/7.88.1','127.0.0.1','2026-11-07 07:32:35.823',NULL,'2026-10-31 07:32:35.823','2026-10-31 07:32:35.824'),(119,'832dc666e389efd30db0e1f4f2683dc6',1,'f51abe29524c2bcabd529017d0cd762c1e35b421d0d5b054c990c68dd0fa9599','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) TraeCN/1.107.1 Chrome/142.0.7444.235 Electron/39.2.7 Safari/537.36','127.0.0.1','2026-11-07 07:37:34.693',NULL,'2026-10-31 07:37:34.693','2026-10-31 07:37:34.693'),(120,'0c7a4b6552fa548673505e05c9c75d47',6,'76be7483e65a6f18d3a17d060e14c9871eb99f5af302938d90f28cdeec348ce9','Go-http-client/1.1','127.0.0.1','2026-11-07 07:45:18.462',NULL,'2026-10-31 07:45:18.462','2026-10-31 07:45:18.462'),(121,'c1a3abd55cf43cfe46e405d5e0dd57f2',6,'0506368373b3fbcb8944134ca32c7dbbe9bcb0377b73642c9c48b4946fd7fcdf','Go-http-client/1.1','127.0.0.1','2026-11-07 07:45:23.076',NULL,'2026-10-31 07:45:23.076','2026-10-31 07:45:23.077'),(122,'eac6541da97ce91bbc3d47b9b0ce390f',1,'65f8f68a1a1bee5b9da2418ea3afaf3afe9d8a581db5164306a537397c1ec91d','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 07:49:22.489',NULL,'2026-10-31 07:49:22.489','2026-10-31 07:49:22.489'),(123,'6d7fd63e0b06ec400f060e74fd19578b',1,'6ed61e655f2fa017a515e2731c4d3d7b330133212744f930c607c13230182d26','curl/7.88.1','127.0.0.1','2026-11-07 07:49:49.168',NULL,'2026-10-31 07:49:49.168','2026-10-31 07:49:49.168'),(124,'45c5710e531a9803ad0d12b1bd38774c',1,'63fb5ab3e62a4530e06a2e871d2daec066b3899020ce19904bfbce7ba32bd0dc','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) TraeCN/1.107.1 Chrome/142.0.7444.235 Electron/39.2.7 Safari/537.36','127.0.0.1','2026-11-07 07:54:26.005',NULL,'2026-10-31 07:54:26.005','2026-10-31 07:54:26.005'),(125,'d38a1dd6428fc00f1cd56375d8ede876',6,'b164ef18361ba3085d75d7f67aa79084b6a0e82a180dfce8d6ae5f6c83bb048e','Go-http-client/1.1','127.0.0.1','2026-11-07 07:59:19.462',NULL,'2026-10-31 07:59:19.462','2026-10-31 07:59:19.462'),(126,'f497aea5952e581d6fa86c872dd30c8f',6,'daa0a0f23dad6ca31b634ec3e36227f80c20ca39996029587633ea3a7d3cfd76','Go-http-client/1.1','127.0.0.1','2026-11-07 07:59:24.077',NULL,'2026-10-31 07:59:24.077','2026-10-31 07:59:24.077'),(127,'b2acc22e867aa9d4782db14bc4591bba',1,'8e861ed4342abdfa08113942a9c122e7a7e73c787e635e59896ed6baa65ee35d','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 08:04:41.178',NULL,'2026-10-31 09:09:01.115','2026-10-31 08:04:41.178'),(128,'6f4090d83b8377aa02691f350e2c580f',6,'1a6f2b348de2d022cc4bae4b955e347772adb006feed06e3345b730b6a875d8d','Go-http-client/1.1','127.0.0.1','2026-11-07 08:13:20.462',NULL,'2026-10-31 08:13:20.462','2026-10-31 08:13:20.462'),(129,'ea7b5e978c16c9f025e5fe65db546440',6,'9f01f31174a09fdc2a1116f66b456620858b858e0ff0d649dd6925f4d04c8155','Go-http-client/1.1','127.0.0.1','2026-11-07 08:13:25.076',NULL,'2026-10-31 08:13:25.076','2026-10-31 08:13:25.076'),(130,'f56c4eb2e49407cf9b4f9de532e678a5',1,'adc3b897bfe6e8ed08ef310bfa69d6f738485f06f7c0cb29a806e2f223a744e2','curl/7.88.1','127.0.0.1','2026-11-07 08:17:28.652',NULL,'2026-10-31 08:17:28.652','2026-10-31 08:17:28.652'),(131,'9df5340a563402e8b465ec7b28c8bcf7',1,'0a5f3fec66124043294688dcbd8a170d993c1cc61a791e283b9743e4ae4f2a3b','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) TraeCN/1.107.1 Chrome/142.0.7444.235 Electron/39.2.7 Safari/537.36','127.0.0.1','2026-11-07 08:18:28.393',NULL,'2026-10-31 08:18:28.393','2026-10-31 08:18:28.393'),(132,'667e2950f8f0a785113cdc718333f59a',6,'c4708409127038681e5bfddea7e48bb760ec3b9e5b50b7cbaf0e69cffd26bbc8','Go-http-client/1.1','127.0.0.1','2026-11-07 08:27:21.462',NULL,'2026-10-31 08:27:21.462','2026-10-31 08:27:21.462'),(133,'e08de6a2d3373805e7980fccec708f84',6,'2988295e8413f7c89e6bb29d3791b7b9ab7dea0eafe640c898dbb7948e503e13','Go-http-client/1.1','127.0.0.1','2026-11-07 08:27:26.076',NULL,'2026-10-31 08:27:26.076','2026-10-31 08:27:26.076'),(134,'8b64ec5360581c349f935f794f62a3ee',1,'d92e2c00855ddedecfa7493cb94cdfc397a4010d97d7064aa9076abfcc880b62','curl/7.88.1','127.0.0.1','2026-11-07 08:38:59.413',NULL,'2026-10-31 08:38:59.413','2026-10-31 08:38:59.413'),(135,'fcadfdb668d89a673615f6b5bcfd900d',6,'61f47cb28430092dcb60ca7a1ea63ef4517ed6b50f27c7457575c624a5748348','Go-http-client/1.1','127.0.0.1','2026-11-07 08:41:22.461',NULL,'2026-10-31 08:41:22.461','2026-10-31 08:41:22.462'),(136,'f9cf7f6a47f76ef55d2bedc12f5810fe',6,'53960e2c5093409a51de6058dc644b435d5e1b5421b6d762dd721b474a30d879','Go-http-client/1.1','127.0.0.1','2026-11-07 08:41:22.471',NULL,'2026-10-31 08:41:22.471','2026-10-31 08:41:22.471'),(137,'dfd24bf7f0232c6b5bfb26fd83c37c0a',6,'1bb2ec1ae718e3c839e5de79ae217bf1faf2b8fd77bbbfb6d0d1b356f674ef8a','Go-http-client/1.1','127.0.0.1','2026-11-07 08:41:27.076',NULL,'2026-10-31 08:41:27.076','2026-10-31 08:41:27.076'),(138,'2c2160c8f53874484222c18ef56a520f',6,'c68722c2d4bd4283fbfd6bdfad1a103231ab31541ae1e78983f9ca748611b4ab','Go-http-client/1.1','127.0.0.1','2026-11-07 08:41:27.086',NULL,'2026-10-31 08:41:27.086','2026-10-31 08:41:27.086'),(139,'18e5ee69612f2b545621ef13a6777c8b',6,'0e2db0b3dff491d8c22150e3e97777443e91980b51db77d7cb12b84246526723','Go-http-client/1.1','127.0.0.1','2026-11-07 08:55:23.462',NULL,'2026-10-31 08:55:23.462','2026-10-31 08:55:23.462'),(140,'3c050e4857a6a51437f4bbdc60f672fb',6,'2137e133852d101a07186c477483723bed04c69df484705c2ecd57353c1f641a','Go-http-client/1.1','127.0.0.1','2026-11-07 08:55:28.076',NULL,'2026-10-31 08:55:28.076','2026-10-31 08:55:28.076'),(141,'e668df69b96fd9b419c2d2dbc07666d9',1,'6efce2cfac1072b0b1fa3ac7e85641c3738bf2e991b9cdd56431986e7d1e21b9','curl/7.88.1','127.0.0.1','2026-11-07 09:02:03.413',NULL,'2026-10-31 09:02:03.413','2026-10-31 09:02:03.413'),(142,'2ed588f04559f008bddcf1fb449eebaa',1,'4c488bbe0334ee0ea61ad96a4756a84553d2a1829d6cf9c535575afe11557509','curl/7.88.1','127.0.0.1','2026-11-07 09:05:45.318',NULL,'2026-10-31 09:05:45.318','2026-10-31 09:05:45.319'),(143,'7594a24fbb601a2bc77deb5ec38d21c4',6,'e81d2bc3007cadccf9f386e19caca176925c31c78da33fd40bd1654d8cff3551','Go-http-client/1.1','127.0.0.1','2026-11-07 09:09:24.462',NULL,'2026-10-31 09:09:24.462','2026-10-31 09:09:24.462'),(144,'721cc9b218b367ad330803ea297f6ba0',6,'390fb6d0853d1d26ec7d9e290ecc8eb991ecbdd5dd2a616d5bab5c8fdf03fcf7','Go-http-client/1.1','127.0.0.1','2026-11-07 09:09:29.076',NULL,'2026-10-31 09:09:29.076','2026-10-31 09:09:29.076'),(145,'555c91fab9e8aa3e6462eaa6c63c9fa2',6,'005fb9595fdd769f012e2be06e462c2947085bd0470f20f53e3067611a6be9dd','Go-http-client/1.1','127.0.0.1','2026-11-07 09:23:25.462',NULL,'2026-10-31 09:23:25.462','2026-10-31 09:23:25.462'),(146,'bd3685263eac72762e2ab6c2b390d0d9',6,'24453d44240efb7c985137ee4757800a0ebc6a77cb6abe76d2e287c748d57835','Go-http-client/1.1','127.0.0.1','2026-11-07 09:23:30.077',NULL,'2026-10-31 09:23:30.077','2026-10-31 09:23:30.077'),(147,'79bc0d00576070a2f9e335ee06ef0aa4',1,'1697e62ad04b2f78d8831b273f495c9843e96a1ff05d77be6e7ffb53f91059ae','curl/7.88.1','127.0.0.1','2026-11-07 09:27:41.590',NULL,'2026-10-31 09:27:41.590','2026-10-31 09:27:41.590'),(148,'216f21581d713b4f168449386f7413fc',6,'d1de5b7fdac379c031a4f8bc2d736783f80979694048ca384ffc0cefb0569105','Go-http-client/1.1','127.0.0.1','2026-11-07 09:37:26.462',NULL,'2026-10-31 09:37:26.462','2026-10-31 09:37:26.462'),(149,'16bd13042625ab4d72c8e18e22c206a8',6,'53b9e37de44645a031bbf2ddddd15dc764b0db33288fd7fbd2b8afc6f0ccf942','Go-http-client/1.1','127.0.0.1','2026-11-07 09:37:31.076',NULL,'2026-10-31 09:37:31.076','2026-10-31 09:37:31.077'),(150,'f0a027ca8f6077847bbe8825add51e68',6,'883612ebd1fdef77cc5bba5a33ffbf3c9efdb2169901b9247a5f190cdecca25e','Go-http-client/1.1','127.0.0.1','2026-11-07 09:51:27.461',NULL,'2026-10-31 09:51:27.461','2026-10-31 09:51:27.461'),(151,'3a925d2ef02501fedbeceab4e884c539',6,'dc1b58e4780380cbd274cbdb0aa9d115720172f65a425dce3233a292b477ea20','Go-http-client/1.1','127.0.0.1','2026-11-07 09:51:32.076',NULL,'2026-10-31 09:51:32.076','2026-10-31 09:51:32.076'),(152,'4f533ca1fd48d8234d3628e1302a354c',6,'8fdad366d1139ec0705c70d6b5e3707ac4b63ce800c708b850989bcb14e8bf21','Go-http-client/1.1','127.0.0.1','2026-11-07 10:05:28.462',NULL,'2026-10-31 10:05:28.462','2026-10-31 10:05:28.462'),(153,'5950f6fd86a6cfec5a6931ac581fdf4c',6,'5258fd250cae777dcb46d69591d584f3daefea59f7255c14bf170a5148c8c183','Go-http-client/1.1','127.0.0.1','2026-11-07 10:05:33.077',NULL,'2026-10-31 10:05:33.077','2026-10-31 10:05:33.077'),(154,'89d7523b91061a1d0952632d9f58e3f4',6,'97811587f932f7303af2db1f9d32e2fc83c38ec86383371d8de0af99ae988d8e','Go-http-client/1.1','127.0.0.1','2026-11-07 10:19:29.461',NULL,'2026-10-31 10:19:29.461','2026-10-31 10:19:29.461'),(155,'c694f46018a2e1bcba167f2ee00656c8',6,'d5f4641aced9a98888b33be01ee595ea60c75d5340d6607c3b271a147c9c42d4','Go-http-client/1.1','127.0.0.1','2026-11-07 10:19:34.076',NULL,'2026-10-31 10:19:34.076','2026-10-31 10:19:34.076'),(156,'0ab9a48b020defb66141f9f555d1173c',6,'c69af9bffb7999ffa2f6069498ef64ce9cfe4f3c2f2e4edb436e56b64588dd68','Go-http-client/1.1','127.0.0.1','2026-11-07 10:33:30.462',NULL,'2026-10-31 10:33:30.462','2026-10-31 10:33:30.462'),(157,'9def8215c5c828878746c1db324cbe9b',6,'5eb96555abc0b7eb6a08b021bc4e505dfd5a91d4f597cedf809e5bc2c4370303','Go-http-client/1.1','127.0.0.1','2026-11-07 10:33:35.076',NULL,'2026-10-31 10:33:35.076','2026-10-31 10:33:35.076'),(158,'6fd6a1c5ee77201c805833cd0b285628',6,'56ba0dde4a7f99386c6e75a2631d10b802a309cf51b2c2226add364bb30a41b6','Go-http-client/1.1','127.0.0.1','2026-11-07 10:47:31.462',NULL,'2026-10-31 10:47:31.462','2026-10-31 10:47:31.462'),(159,'27290043997fea574e98990959bf123b',6,'a0763769ab18858fe35bec03cf7fca0f9f383b43e5fc9412c427a860c9a5b10e','Go-http-client/1.1','127.0.0.1','2026-11-07 10:47:36.076',NULL,'2026-10-31 10:47:36.076','2026-10-31 10:47:36.076'),(160,'0ea3957ff2a303f92710120ea544c7b7',6,'f39968bb3c3314613b775c6acb2b063f141a2c79e35d7b80d13b094255c5631a','Go-http-client/1.1','127.0.0.1','2026-11-07 11:01:32.462',NULL,'2026-10-31 11:01:32.462','2026-10-31 11:01:32.462'),(161,'a8516fbd2a8022c141347b8907980965',6,'ca92199dc7927350cf03ddc44bb5e7c5fcd07aa88ebb4e0e756dbe774f80972d','Go-http-client/1.1','127.0.0.1','2026-11-07 11:01:37.077',NULL,'2026-10-31 11:01:37.077','2026-10-31 11:01:37.077'),(162,'32aba242397e17e46432c6de537f4086',6,'c5d71bde47ead949c50615c7330304ad122e53d3ac2073276daf3133ea721a6a','Go-http-client/1.1','127.0.0.1','2026-11-07 11:15:33.462',NULL,'2026-10-31 11:15:33.462','2026-10-31 11:15:33.462'),(163,'921255662168c32d786d33d4fb8ee6a9',6,'1919be8040b035b8fa9463c9fcdd9a2bbd3e5571b533e7b26379ca6c5d8f9220','Go-http-client/1.1','127.0.0.1','2026-11-07 11:15:38.076',NULL,'2026-10-31 11:15:38.076','2026-10-31 11:15:38.076'),(164,'3c6bac28a0b6a946ca74220303edc7c6',1,'72eef301fbb1185d9d5b707f33260bb878d8f0aea0880460ee889bd2b5c7d77a','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 11:17:31.179',NULL,'2026-10-31 12:36:45.650','2026-10-31 11:17:31.179'),(165,'46063790bb0b52935e74db3ad3fd1e7d',1,'46b1c12705601b34831bab0ed35e2755c0000073f46f9c95a80fbabfce487eb2','curl/7.88.1','127.0.0.1','2026-11-07 11:26:56.724',NULL,'2026-10-31 11:26:56.724','2026-10-31 11:26:56.724'),(166,'3459dd3bc831a763fe67a199f8611a67',6,'bd19423b4bc55a6efa4077e950d3b97c0fcdacf73931c966a9c98d8246d2d01d','Go-http-client/1.1','127.0.0.1','2026-11-07 11:29:34.462',NULL,'2026-10-31 11:29:34.462','2026-10-31 11:29:34.462'),(167,'3ee4910cbe3905f36c9778078fdde914',6,'50d5ecdd91455240bb3b0a8814c13df1c920e7671d846505fe1967b98bf10210','Go-http-client/1.1','127.0.0.1','2026-11-07 11:29:39.077',NULL,'2026-10-31 11:29:39.077','2026-10-31 11:29:39.077'),(168,'151f1e82cdaddaa7e0ef3c7f0547928a',1,'c7da641b6a88a85f70721d879eaae28e4557a24e9fa64ec7e18382fdc28a1d19','curl/7.88.1','127.0.0.1','2026-11-07 11:31:24.911',NULL,'2026-10-31 11:31:24.911','2026-10-31 11:31:24.912'),(169,'da1e909bbf0303f990c9fb5370f0e763',1,'70f8539c2f66fe72429a79c0c40c2f040067e8531aaa30068bf49903761ddfe7','curl/7.88.1','127.0.0.1','2026-11-07 11:43:09.360',NULL,'2026-10-31 11:43:09.360','2026-10-31 11:43:09.360'),(170,'6de99d12e3195f687d041ad96888fc0d',6,'73386efe305b54e4f4d54887adafe5a5348a2a2467db568b38f62cade7dd1848','Go-http-client/1.1','127.0.0.1','2026-11-07 11:43:35.462',NULL,'2026-10-31 11:43:35.462','2026-10-31 11:43:35.462'),(171,'dc18ef653104583c3a269675fdbce70b',6,'3a501f3b9ebba206a82430bc88562579302b1688fb527b65bc7fe199da2a5f6c','Go-http-client/1.1','127.0.0.1','2026-11-07 11:43:40.078',NULL,'2026-10-31 11:43:40.078','2026-10-31 11:43:40.078'),(172,'5687fdada62f32c5babfbd89f89d45f9',6,'53bcfda4df89b4b2d0ab6bccfdf5f14fcb1ac11c4572556e77b9a15fc289d59d','Go-http-client/1.1','127.0.0.1','2026-11-07 11:53:57.317',NULL,'2026-10-31 11:53:57.317','2026-10-31 11:53:57.317'),(173,'3e764e51ccf60b4efec08bbe60689210',6,'3e8441dfe23931095322cd58d81dd2c32948dc06f5a7ca5e358298bb8418a092','Go-http-client/1.1','127.0.0.1','2026-11-07 11:53:59.283',NULL,'2026-10-31 11:53:59.283','2026-10-31 11:53:59.283'),(174,'1f93b0a185710fc8f57b8d6065bdd8d1',6,'33da5d88a4380e6cc16e2d476145686a59259023c0ddaecd24cef78dc9441814','Go-http-client/1.1','127.0.0.1','2026-11-07 11:55:04.559',NULL,'2026-10-31 11:55:04.559','2026-10-31 11:55:04.560'),(175,'7935d5a9192fbcc9982440e33a45b71e',6,'24319e38283769e241d520d648ac4ec6eaeddaad16d193200cf110f2b75b00f5','Go-http-client/1.1','127.0.0.1','2026-11-07 11:58:13.768',NULL,'2026-10-31 11:58:13.768','2026-10-31 11:58:13.768'),(176,'5ada79ed7be337f89d6e04783d42d7ce',1,'0a33ddf37092e24773dc941ce87cbcd67e4779ede28f341e044a1d78939b8f03','curl/7.88.1','127.0.0.1','2026-11-07 11:58:30.956',NULL,'2026-10-31 11:58:30.956','2026-10-31 11:58:30.956'),(177,'e65267c45d8c09b33d2170e21f836388',6,'299a719366fc5d47dba7dfd2e4ffa1a7a46627815df8d759ead0e5a95d4f7be3','Go-http-client/1.1','127.0.0.1','2026-11-07 12:07:33.059',NULL,'2026-10-31 12:07:33.059','2026-10-31 12:07:33.059'),(178,'6b908cb75d4898d522de76480252ec3b',6,'50e4c2769df64ed7e548eac6fb10b6dde7a2546a99d9a15128e12f9c041dcf46','Go-http-client/1.1','127.0.0.1','2026-11-07 12:07:35.256',NULL,'2026-10-31 12:07:35.256','2026-10-31 12:07:35.256'),(179,'6fbb66eabb784cfe8238fb4fb2b31816',1,'ae81631e7f6e71393cd4b6384013b96bbf3bed73504cac4a4875a4fda53b5e76','curl/7.88.1','127.0.0.1','2026-11-07 12:07:43.039',NULL,'2026-10-31 12:07:43.039','2026-10-31 12:07:43.039'),(180,'8ad3af2440b349dc408a81654041ce28',6,'020a506868adeca0282c31da38c87d1896995aedc972019ee70fdac2035eef24','Go-http-client/1.1','127.0.0.1','2026-11-07 12:14:10.076',NULL,'2026-10-31 12:14:10.076','2026-10-31 12:14:10.076'),(181,'a426b7c2a9f23e44331d1dd4dc7a1e17',6,'b7fec48211d5a6294708bf4c65444c77f70b201471ada03795dff514eec22761','Go-http-client/1.1','127.0.0.1','2026-11-07 12:14:12.260',NULL,'2026-10-31 12:14:12.260','2026-10-31 12:14:12.260'),(182,'16db6c11fdfa8650df338bec33ce986e',1,'41a7e7635cfda759ac9bb63e2fe0d51fe041243cb963783f71629c59bad437ca','curl/7.88.1','127.0.0.1','2026-11-07 12:14:25.129',NULL,'2026-10-31 12:14:25.129','2026-10-31 12:14:25.129'),(183,'80ba25afbac985c33dc18f419e795271',1,'d597c92835bc9bd8967227c1420b1177840403eda9e70f395c817f521d6d7973','curl/7.88.1','127.0.0.1','2026-11-07 12:20:51.298',NULL,'2026-10-31 12:20:51.298','2026-10-31 12:20:51.298'),(184,'c66013841a1e8ee8a7f1429904e9e536',6,'715b099ec24399288837d3377455760ecbd6e926567989a668c774368ae78adb','Go-http-client/1.1','127.0.0.1','2026-11-07 12:27:01.037',NULL,'2026-10-31 12:27:01.037','2026-10-31 12:27:01.037'),(185,'9c3241b1576cc5941ff2ed7abd0c8753',6,'00945d551363391767b58433a7b1b2bc145ab03bc5c8ad341db6680d8fc76b3d','Go-http-client/1.1','127.0.0.1','2026-11-07 12:27:03.255',NULL,'2026-10-31 12:27:03.255','2026-10-31 12:27:03.255'),(186,'1fba3f259dc11d8c0dfa16caefafcb56',1,'b71508e328ac28f095f70072ba8a3c1c43f1f01e0d953009f8fb610cade39dc3','curl/7.88.1','127.0.0.1','2026-11-07 12:27:59.021',NULL,'2026-10-31 12:27:59.021','2026-10-31 12:27:59.022'),(187,'5b55c7d9e2c1ceef0ab0e752c25ef773',6,'d7380802b35da77c7402bef9f14f49763aa0e5ee3e68edc282763af09cb763da','Go-http-client/1.1','127.0.0.1','2026-11-07 12:41:01.802',NULL,'2026-10-31 12:41:01.802','2026-10-31 12:41:01.802'),(188,'8405e565cd21d8e805f482ac0bfbdd75',6,'b2f6e6b0d35818a3df51107fa6786d578f025813f6392babb7087133f6a044e0','Go-http-client/1.1','127.0.0.1','2026-11-07 12:41:04.015',NULL,'2026-10-31 12:41:04.015','2026-10-31 12:41:04.015'),(189,'d73d4044b800167c9b86322e06c4b1f2',6,'4dbe39c00ebe46314fff47419c9692f831416783f0939eebad7fdb2359731d86','Go-http-client/1.1','127.0.0.1','2026-11-07 12:48:53.714',NULL,'2026-10-31 12:48:53.714','2026-10-31 12:48:53.714'),(190,'5038a7d191f62e1a74e5d9a97247f7f6',6,'f6ed119c392f6dbc020a17d03752b765c7cdd06165298e662ad331b3fbbf75af','Go-http-client/1.1','127.0.0.1','2026-11-07 12:48:55.868',NULL,'2026-10-31 12:48:55.868','2026-10-31 12:48:55.868'),(191,'2e6141db48b280885a945cdaab6ee25a',6,'405c8922b4089542ad49029fc154e40e518af7ccc4912bb36bbed504f56453d8','Go-http-client/1.1','127.0.0.1','2026-11-07 12:48:58.013',NULL,'2026-10-31 12:48:58.013','2026-10-31 12:48:58.013'),(192,'419400c4925970d13ae9ca9a4807df9f',6,'c5a308c9ef7f5d43031a04ece188ae1891baceea1a5adef5db04e2ece12350b3','Go-http-client/1.1','127.0.0.1','2026-11-07 12:51:05.482',NULL,'2026-10-31 12:51:05.482','2026-10-31 12:51:05.482'),(193,'f23480cecccc40c86893b976a40d11d2',6,'4b73ce1fc380157517fae1083520914035768ae9678a7a5801df1283a24ccd10','Go-http-client/1.1','127.0.0.1','2026-11-07 12:51:05.635',NULL,'2026-10-31 12:51:05.635','2026-10-31 12:51:05.635'),(194,'dadb9a4c3d855bc172970039d6f3b7a9',6,'0eed4ed9eed61b558cf1f96816d1200681d49a07b79a7e9b073e09e3bb9dc6f3','Go-http-client/1.1','127.0.0.1','2026-11-07 12:51:05.780',NULL,'2026-10-31 12:51:05.780','2026-10-31 12:51:05.780'),(195,'aefb220a30fa76e61a0411561f51487c',1,'b58502ae75cb0c9a2f2463d8546478aca89bd1690d2495cf1f11931c0195d8b3','curl/7.88.1','127.0.0.1','2026-11-07 12:51:50.244',NULL,'2026-10-31 12:51:50.244','2026-10-31 12:51:50.244'),(196,'22c9259eb724ab291b69ad116c5d5858',1,'5a823e44f085f93c703a96ae00d951e62318fd10a4d13902367fabd4363aefcc','curl/7.88.1','127.0.0.1','2026-11-07 12:52:04.041',NULL,'2026-10-31 12:52:04.041','2026-10-31 12:52:04.041'),(197,'13d9a6116d9688aff570e455326d87bf',6,'2e04bbaaa3e96cd62bf67b12f965a6a73e7dc6ef96ef29d440a6e86152d6fac3','Go-http-client/1.1','127.0.0.1','2026-11-07 12:52:44.638',NULL,'2026-10-31 12:52:44.638','2026-10-31 12:52:44.638'),(198,'1074beb2b853d7380606929f8b4b4e35',6,'a73b0ce63618e7675d9e637fb51e08424b4818670002a7e6146d9541ffab3a22','Go-http-client/1.1','127.0.0.1','2026-11-07 12:55:12.954',NULL,'2026-10-31 12:55:12.954','2026-10-31 12:55:12.954'),(199,'fcec080bc71e0d68e1a12198ccf737b4',6,'3e3066ee19aaf07d477d3a36f7b97b630efd61b2bb6fcacc8edcf5ddcfb4dd3e','Go-http-client/1.1','127.0.0.1','2026-11-07 12:57:26.616',NULL,'2026-10-31 12:57:26.616','2026-10-31 12:57:26.616'),(200,'e842caf83383d1b78f47556a8b38426c',6,'74a7f48b6aa810efea2811a8754a3c840cc11c4540c977190b4168103fe378f0','Go-http-client/1.1','127.0.0.1','2026-11-07 12:58:10.124',NULL,'2026-10-31 12:58:10.124','2026-10-31 12:58:10.125'),(201,'479a59d3adcff4c965a53db6cdf9a09c',6,'ebb552064260d03028035353f20c33ed48c812cc67d55d456748e4eeefd2573c','Go-http-client/1.1','127.0.0.1','2026-11-07 13:02:39.462',NULL,'2026-10-31 13:02:39.462','2026-10-31 13:02:39.462'),(202,'ede09db8b350457f720dfd7a6c4ea370',6,'fa3015e50f17a4b36ba1a09eb319b742e25c547b19717f5e7c4ed16d7c2cd877','Go-http-client/1.1','127.0.0.1','2026-11-07 13:03:05.430',NULL,'2026-10-31 13:03:05.430','2026-10-31 13:03:05.430'),(203,'af7d3f8f9cd64f17cd991501aff460ee',1,'38300d7e9c935f47231610837534225cd674080bb5c0d3b77070e77db030da80','curl/7.88.1','127.0.0.1','2026-11-07 13:03:53.737',NULL,'2026-10-31 13:03:53.737','2026-10-31 13:03:53.737'),(204,'aceba2a5421bc8d2642bdaa0a7e6465b',1,'dcc2408ea190785e59dd14094436ba873939b845524192ebe7b42e11bfd478cc','curl/7.88.1','127.0.0.1','2026-11-07 13:03:58.057',NULL,'2026-10-31 13:03:58.057','2026-10-31 13:03:58.057'),(205,'7c4764069d7dff1b6f23736a66f29a7c',6,'47273df0d7f9c89c9aa36a485749d9369c37528a740bae0d66fd44f1ae42cc62','Go-http-client/1.1','127.0.0.1','2026-11-07 13:05:06.637',NULL,'2026-10-31 13:05:06.637','2026-10-31 13:05:06.637'),(206,'005aa336e5dcedee424d7b112e21e1e2',1,'878f21a6df9379bfc296350a85b96e665351efd40e9bb8488bd7cb619ab7f07f','curl/7.88.1','127.0.0.1','2026-11-07 13:09:30.563',NULL,'2026-10-31 13:09:30.563','2026-10-31 13:09:30.563'),(207,'c092b2cf94108cffced2b1d1265872d9',6,'1757083d5e68a27db8a638bad2e1eaba330e447f96e635bc0a445886c8fdd356','Go-http-client/1.1','127.0.0.1','2026-11-07 13:16:40.224',NULL,'2026-10-31 13:16:40.224','2026-10-31 13:16:40.224'),(208,'06db2b27c2ec850c754b7388f4682b54',6,'5a8dfc918213cffbccd0a1f16b18c5efc19cf8e22d84c857216370cc42f34241','Go-http-client/1.1','127.0.0.1','2026-11-07 13:19:07.634',NULL,'2026-10-31 13:19:07.634','2026-10-31 13:19:07.634'),(209,'facb2e60f94121cb3ba51b1954331c1f',6,'22c2ec3177d2a9bce9049401b87a9650886c2c4e6d673d763da27ae3bc804486','Go-http-client/1.1','127.0.0.1','2026-11-07 13:30:41.223',NULL,'2026-10-31 13:30:41.223','2026-10-31 13:30:41.223'),(210,'9e6eb83d59c8f4fefab79c101ab205d7',6,'529e5f2ea4a82a082a48863aab9939b180669a589c548e23f101b7bf7ad358c1','Go-http-client/1.1','127.0.0.1','2026-11-07 13:33:08.634',NULL,'2026-10-31 13:33:08.634','2026-10-31 13:33:08.634'),(211,'0d58f50d950f470ee4a23d54443c7852',6,'ba225d5b70563846c8d2700295eb5dbda64b2e75b16d297ff73be22a8abc2c02','Go-http-client/1.1','127.0.0.1','2026-11-07 13:44:42.223',NULL,'2026-10-31 13:44:42.223','2026-10-31 13:44:42.223'),(212,'e595363608c5dcc1b8e21d1a1cdbe184',6,'2103f60599ac498ef414a98a3bb9721de940921872c359612aa4c555f91248b9','Go-http-client/1.1','127.0.0.1','2026-11-07 13:47:09.635',NULL,'2026-10-31 13:47:09.635','2026-10-31 13:47:09.635'),(213,'1f288abc76d02f837773aec3c2521df9',6,'38343c99b108e9d9e0d77b1d504b575b0e654750c38ed77a50a507190fdc0a99','Go-http-client/1.1','127.0.0.1','2026-11-07 13:58:43.224',NULL,'2026-10-31 13:58:43.224','2026-10-31 13:58:43.224'),(214,'4345483aa287546f5bf3fadad29b63b1',6,'31c7a1eebbad605f5bd6c84a77bfa5dbebd36113b30f19db48c688c16b75d12e','Go-http-client/1.1','127.0.0.1','2026-11-07 14:01:10.636',NULL,'2026-10-31 14:01:10.636','2026-10-31 14:01:10.636'),(215,'4ea5d3c447968c715fa3905ceeffbf34',1,'f7661e786673775e449034036a1db1680b2e8bd69455a650c813e5b11e744188','curl/7.88.1','127.0.0.1','2026-11-07 14:01:12.342',NULL,'2026-10-31 14:01:12.342','2026-10-31 14:01:12.342'),(216,'f95ed7d249513e60b5b06571dea0452b',1,'002d4c387b635b3d718c17912bd4d672320ac070356ca81e1f27188bd7dd7305','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','127.0.0.1','2026-11-07 14:01:24.306',NULL,'2026-10-31 14:01:24.306','2026-10-31 14:01:24.306'),(217,'da5dbb335be28402052b6c9c5c5ad3c6',1,'3904280a89725890ae38158650fdd6e27cbb767f3ca07b4778b0cef9fdc799f5','curl/7.88.1','127.0.0.1','2026-11-07 14:06:50.808',NULL,'2026-10-31 14:06:50.808','2026-10-31 14:06:50.808'),(218,'9dbe9183a8481e0459239fcebf6c49fc',1,'d334e9025f4e6ee0389a36fc23a370a9e8d0235b6efa4c31fa8aaea7b5b2b767','curl/7.88.1','127.0.0.1','2026-11-07 14:07:02.799',NULL,'2026-10-31 14:07:02.799','2026-10-31 14:07:02.799'),(219,'42cc1d8d8f9685e7f2cf60c9764cd606',6,'6c3ce89c375713d1d9912b9986e874a03b66049e7b4ce9bc8a61c987b62b658d','Go-http-client/1.1','127.0.0.1','2026-11-07 14:12:44.224',NULL,'2026-10-31 14:12:44.224','2026-10-31 14:12:44.224'),(220,'97f074d292315d5d8ee2d1acadc4479f',1,'39d3616233e9fab0aa8fcb867b3ed623aa48e08046042321edfe2bff3de32e2a','curl/7.88.1','127.0.0.1','2026-11-07 14:13:09.660',NULL,'2026-10-31 14:13:09.660','2026-10-31 14:13:09.660'),(221,'54bcbb252ef4bfc2f16f8da80ab45662',1,'ccd1643c64c8b8a0676cc3be703fddc898a2d45c3d745612172b5ac5fbe36e20','curl/7.88.1','127.0.0.1','2026-11-07 14:13:59.535',NULL,'2026-10-31 14:13:59.535','2026-10-31 14:13:59.535'),(222,'87741a6481bc7a11d1ef2e0ad7710721',6,'3751d56aa55f40a37b164c655f19b372b8c7006137756e6f00a9438c5341969b','Go-http-client/1.1','127.0.0.1','2026-11-07 14:15:10.872',NULL,'2026-10-31 14:15:10.872','2026-10-31 14:15:10.872'),(223,'74645522ae43cc6f4e6065c42c278f6c',1,'a2f338dcc970e523f6980ac2f8d60a6d6030ba181d9549cca4f035552c03341a','curl/7.88.1','127.0.0.1','2026-11-07 14:19:28.322',NULL,'2026-10-31 14:19:28.322','2026-10-31 14:19:28.323'),(224,'a036399bcc1aef4b3176f35dce2c640c',1,'65347fb2931b9185a43db305b837dc1556a74bb522391cef1f629231a2d93c8d','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','182.139.93.128','2026-11-07 14:19:41.615',NULL,'2026-10-31 14:19:41.615','2026-10-31 14:19:41.615'),(225,'69a59a7a563de95fd2fcca2074465790',1,'f1e60326d71c1101894d00e62b2ef840c63928492efb8a597f0475a82cb46798','curl/7.88.1','127.0.0.1','2026-11-07 14:19:41.672',NULL,'2026-10-31 14:19:41.672','2026-10-31 14:19:41.672'),(226,'2246f9c21380dc79f56fb983333bb586',1,'6fba80a424549c6a38e1cadcd9e5fd6bd880faa8f747603a1b5fb7650e91fb17','curl/7.88.1','129.226.185.231','2026-11-07 14:24:22.172',NULL,'2026-10-31 14:24:22.172','2026-10-31 14:24:22.173'),(227,'640d59fcf47b3f71127ae6230ede48e7',1,'a8ac0c2cf3c5d7bc3b097613f2a5eedda2af09065b62fea65dac57930902ce75','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','182.139.93.128','2026-11-07 14:24:44.986',NULL,'2026-10-31 14:24:44.986','2026-10-31 14:24:44.986'),(228,'687f3fbf7369c2c4bde556d21621c08b',1,'3ef07ddb86ceb9bbc0cecc2ac734aaa9c6340e7df85e70a6eb8f5dad97e8f23d','curl/7.88.1','129.226.185.231','2026-11-07 14:28:17.160',NULL,'2026-10-31 14:28:17.160','2026-10-31 14:28:17.160'),(229,'4c911bb4f093ec5f3e95706f07fd7f29',1,'eb3af817997ec4851661a3bfd28b0f13e8ae255f54f9ef4676b17a9ac8f0edaf','curl/7.88.1','129.226.185.231','2026-11-07 14:37:33.670',NULL,'2026-10-31 14:37:33.670','2026-10-31 14:37:33.670'),(230,'52b28954404edc9879a8502e2141a9b8',1,'fb2e3eaf79a06d7f63ac71eff37daba0857bfaa91da76c4d11e6e8366979f043','curl/7.88.1','129.226.185.231','2026-11-07 14:38:06.503',NULL,'2026-10-31 14:38:06.503','2026-10-31 14:38:06.503'),(231,'126be5a6e83dc7eb610d22c934b812af',1,'dbb63aecd7d3edf00356990f3c854d26d11bf62290cb614389f45f5cfd1d52bb','curl/7.88.1','129.226.185.231','2026-11-07 14:38:26.719',NULL,'2026-10-31 14:38:26.719','2026-10-31 14:38:26.719'),(232,'efd27ecf09aefad4acb70dc392ece6a4',6,'f79da59c2d28ff17392aaa16053bd4faf87b5a9287aefddf2173d8186c920c15','Go-http-client/2.0','129.226.185.231','2026-11-07 14:38:40.807',NULL,'2026-10-31 14:38:40.807','2026-10-31 14:38:40.807'),(233,'76a144fad597e8bc90b672b28f380c58',1,'5bca782a8209566937f07d72637024304e34f85182c16285492cea3f39358230','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','182.139.93.128','2026-11-07 14:40:50.825',NULL,'2026-10-31 14:40:50.825','2026-10-31 14:40:50.825'),(234,'073bf9bb1ea6b3112b50166b72869c2b',6,'9b20a3e45dedff5c8a27a0b3433b164fbf6ff64f76a3921d1abc453b50731735','Go-http-client/2.0','132.232.177.36','2026-11-07 14:41:31.116',NULL,'2026-10-31 14:41:31.116','2026-10-31 14:41:31.116'),(235,'af99ec8392975e380def70f56ffeeec4',1,'0a2d967af88f30565aa21c348328b6aaee42bd6de66f26ca0ddddbf529652497','curl/7.88.1','129.226.185.231','2026-11-07 14:44:57.762',NULL,'2026-10-31 14:44:57.762','2026-10-31 14:44:57.762'),(236,'469af566773a2b891696da8d7370c59d',1,'36916ec9562e911526259140a4516930595ed5bec296dabc5d251e9256f3ed02','curl/7.88.1','129.226.185.231','2026-11-07 14:45:16.806',NULL,'2026-10-31 14:45:16.806','2026-10-31 14:45:16.806'),(237,'2163654d7eb40c7a62b6df0cfadfa42e',1,'682130c34eb0cec511f7fffad1c83fe809cf602f2da3a198a7d560d604588ad8','curl/7.88.1','129.226.185.231','2026-11-07 14:49:26.996',NULL,'2026-10-31 14:49:26.996','2026-10-31 14:49:26.996'),(238,'4cdc6141786a547f8f0891722d1c47e5',1,'f4e3843a9069c6ace8c007a6d12b0377028786b328ce5d1b31002c042818b770','curl/7.88.1','129.226.185.231','2026-11-07 14:50:29.154',NULL,'2026-10-31 14:50:29.154','2026-10-31 14:50:29.154'),(239,'e10e083bf4aefdf592ad4c45bd871032',1,'c175ca2cf8b1234b2a4b19ea424320a3dc92d536d1e8753e728f88aacd2cfa92','curl/7.88.1','129.226.185.231','2026-11-07 14:50:36.589',NULL,'2026-10-31 14:50:36.589','2026-10-31 14:50:36.589'),(240,'33d9108573f82b29c0bb20e826be5d8f',1,'971c8ec55decb3f4b14e835eb3f003c7d46b310a4035e994e8da422ae886b634','curl/7.88.1','129.226.185.231','2026-11-07 14:50:42.860',NULL,'2026-10-31 14:50:42.860','2026-10-31 14:50:42.861'),(241,'70bead7b179456011f545b6c819adde3',1,'bc01abaeb6d4421c948e85b287663aea4ec50874f6545f460ea44757c87b3e72','curl/7.88.1','129.226.185.231','2026-11-07 14:54:47.815',NULL,'2026-10-31 14:54:47.815','2026-10-31 14:54:47.815'),(242,'7801600f074deb4d1b5fb25bbae95f3a',6,'aeb3c73ca58056a50e322121987f7ce58ec6d094d987d900b09b0351df25ad12','Go-http-client/2.0','132.232.177.36','2026-11-07 14:55:31.304',NULL,'2026-10-31 14:55:31.304','2026-10-31 14:55:31.305'),(243,'376bdb3f96450eaa83f3980d5d638201',1,'cfa505fe779731a9a590e37b3fdf339265cd3f178bc138d2dbfa28c012c7ce82','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) TraeCN/1.107.1 Chrome/142.0.7444.235 Electron/39.2.7 Safari/537.36','182.139.93.128','2026-11-07 14:56:38.912',NULL,'2026-10-31 14:56:38.912','2026-10-31 14:56:38.912'),(244,'151e1cb5bf2ec9acc9c6325eb3602d77',1,'ec82af63ab7cd6a9dccdb5e928b780c99e4628e49c4f3bf8ae12f018b3807772','Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36','182.139.93.128','2026-11-07 14:57:24.237',NULL,'2026-10-31 15:29:54.841','2026-10-31 14:57:24.237'),(245,'4ff49f08741474a1281d15edb9cd9064',6,'403797b1e3adfd63e18f45a076a71c2b1d42b3bdc4794a1fe706f9a7167e83c1','Go-http-client/2.0','132.232.177.36','2026-11-07 15:09:31.829',NULL,'2026-10-31 15:09:31.829','2026-10-31 15:09:31.829'),(246,'f873147f3c39bfaa8a9e597465131e76',1,'a2cab9b2b234b65119aa049ca0effb8216582b83a9de43948f6219fc6fc74f9d','curl/7.88.1','129.226.185.231','2026-11-07 15:09:41.569',NULL,'2026-10-31 15:09:41.569','2026-10-31 15:09:41.569'),(247,'fdb5bfbbfb5919e076ceff2f0a3b5411',12,'721143b63ddbc3bf825ee8cec56c00b91119f745713f9d83fb010eb8ed11b17c','curl/7.88.1','129.226.185.231','2026-11-07 15:10:30.115',NULL,'2026-10-31 15:10:30.115','2026-10-31 15:10:30.115'),(248,'996c8df7a92104e30fb4ae26a69b1101',6,'165f3ed35687dbdb7f66c95aaeb34664f0dff05d1ddc7c4ee7e814c3e5bf551d','Go-http-client/2.0','132.232.177.36','2026-11-07 15:23:32.829',NULL,'2026-10-31 15:23:32.829','2026-10-31 15:23:32.829'),(249,'32c5ee3d840837922c58dcdae0230b85',1,'c88dcc40ff64e652a94932546b8ff41f47770ae34090794579310f2265047c4a','curl/7.88.1','129.226.185.231','2026-11-07 15:32:16.280',NULL,'2026-10-31 15:32:16.280','2026-10-31 15:32:16.280'),(250,'145dde166ea09753fd689c6f4d849ced',12,'aecd5c8bb7dd077752e51f72438ed361cc89007de4592d48615c792175ce2068','curl/7.88.1','129.226.185.231','2026-11-07 15:32:16.445',NULL,'2026-10-31 15:32:16.445','2026-10-31 15:32:16.445'),(251,'2f045b7ff9d4a4297acb0cf7feec790d',6,'fda1d0a9f9703b1cc24db7411f73902efe2a50e9caa898dc1c870fe452fa36d5','Go-http-client/2.0','132.232.177.36','2026-11-07 15:37:33.833',NULL,'2026-10-31 15:37:33.833','2026-10-31 15:37:33.833'),(252,'75f6f3e6e35df88b9d474fcd09b33f29',6,'9aea067b97dd47fdb46b8c84ac17dc2d3ae8761d8dc97a9d2ace5bedc3f394c2','Go-http-client/2.0','132.232.177.36','2026-11-07 15:51:34.833',NULL,'2026-10-31 15:51:34.833','2026-10-31 15:51:34.834'),(253,'15066bd216e4bb15e2aaa1df1b0379cd',6,'a9816e13815303ce3acda5a67f2350282921f5e867e9c101bd3915e30da7e570','Go-http-client/2.0','132.232.177.36','2026-11-07 16:05:35.835',NULL,'2026-10-31 16:05:35.835','2026-10-31 16:05:35.835');
/*!40000 ALTER TABLE `sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `site_configs`
--

DROP TABLE IF EXISTS `site_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `site_configs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `site_title` varchar(128) NOT NULL,
  `site_subtitle` varchar(255) NOT NULL,
  `register_open` tinyint(1) NOT NULL DEFAULT '0',
  `register_review` tinyint(1) NOT NULL DEFAULT '1',
  `distribution_open` tinyint(1) NOT NULL DEFAULT '0',
  `distribution_ratio` decimal(6,2) NOT NULL DEFAULT '0.00',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `announcement` varchar(512) NOT NULL DEFAULT '',
  `public_url` varchar(255) NOT NULL DEFAULT '',
  `agent_download_url` varchar(255) NOT NULL DEFAULT '',
  `agent_broker` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `site_configs`
--

LOCK TABLES `site_configs` WRITE;
/*!40000 ALTER TABLE `site_configs` DISABLE KEYS */;
INSERT INTO `site_configs` VALUES (1,'Aoe-tunnel','控制台入口',1,0,1,20.00,'2026-10-31 01:53:24.792','2026-10-31 15:09:41.609','','https://ask.30ss.vip','','ssl://ask.30ss.vip:8883');
/*!40000 ALTER TABLE `site_configs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `traffic_stats`
--

DROP TABLE IF EXISTS `traffic_stats`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `traffic_stats` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `rule_id` bigint unsigned NOT NULL,
  `node_code` varchar(64) NOT NULL,
  `up_bytes` bigint NOT NULL,
  `down_bytes` bigint NOT NULL,
  `bill_bytes` bigint NOT NULL,
  `recorded_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_rule_time` (`rule_id`,`recorded_at`)
) ENGINE=InnoDB AUTO_INCREMENT=73 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `traffic_stats`
--

LOCK TABLES `traffic_stats` WRITE;
/*!40000 ALTER TABLE `traffic_stats` DISABLE KEYS */;
INSERT INTO `traffic_stats` VALUES (1,1,'n-entry',1024,8192,9216,'2026-10-30 19:50:55.989'),(2,9,'n-entry',87,1048781,1048868,'2026-10-30 20:30:20.338'),(3,9,'n-entry',87,1048781,1048868,'2026-10-30 20:30:26.338'),(4,10,'n-entry',87,1048781,1048868,'2026-10-30 20:30:48.338'),(5,1,'n-entry',87,0,87,'2026-10-30 20:34:42.338'),(6,11,'n-entry',87,1048781,1048868,'2026-10-30 20:35:12.338'),(7,11,'n-entry',87,1048781,1048868,'2026-10-30 20:35:34.339'),(8,11,'n-entry',87,1048781,1048868,'2026-10-30 20:42:18.339'),(9,11,'n-entry',87,1048781,1048868,'2026-10-30 20:42:26.339'),(10,11,'n-entry',87,1048781,1048868,'2026-10-30 21:28:04.336'),(11,11,'n-entry',87,1048781,1048868,'2026-10-30 21:28:24.336'),(12,11,'n-entry',87,1048781,1048868,'2026-10-30 21:28:32.337'),(13,11,'n-entry',87,1048781,1048868,'2026-10-30 21:28:58.980'),(14,11,'n-entry',87,1048781,1048868,'2026-10-30 21:29:20.237'),(15,11,'n-entry',87,1048781,1048868,'2026-10-30 21:30:11.227'),(16,11,'n-entry',87,1048781,1048868,'2026-10-30 21:38:17.228'),(17,12,'n-entry',87,1048781,1048868,'2026-10-30 21:38:17.228'),(18,14,'n-entry',87,1048781,1048868,'2026-10-30 21:57:34.313'),(19,11,'n-entry',87,1048781,1048868,'2026-10-30 21:57:46.312'),(20,11,'n-entry',87,1048781,1048868,'2026-10-30 21:58:48.312'),(21,11,'n-entry',87,1048781,1048868,'2026-10-30 22:03:23.930'),(22,14,'n-entry',261,3146343,3146604,'2026-10-30 22:03:23.930'),(23,14,'n-entry',87,1048781,1048868,'2026-10-30 22:03:41.931'),(24,14,'n-entry',87,1048781,1048868,'2026-10-30 22:03:59.930'),(25,11,'n-entry',87,1048781,1048868,'2026-10-30 22:03:59.930'),(26,15,'n-in',79,1111,1190,'2026-10-30 23:18:53.378'),(27,11,'n-entry',79,1111,1190,'2026-10-30 23:19:57.278'),(28,12,'n-entry',79,1111,1190,'2026-10-30 23:19:57.278'),(29,14,'n-entry',79,1111,1190,'2026-10-30 23:19:57.278'),(30,15,'n-in',79,1111,1190,'2026-10-30 23:19:57.377'),(31,12,'n-entry',201,1111,1312,'2026-10-30 23:55:17.278'),(32,12,'n-entry',1499,414,1913,'2026-10-30 23:55:37.278'),(33,12,'n-entry',207,1631,1838,'2026-10-30 23:55:39.278'),(34,23,'n-in',79,1111,1190,'2026-10-31 00:46:59.131'),(35,15,'n-in',79,1111,1190,'2026-10-31 01:13:50.962'),(36,15,'n-in',174,544776,544950,'2026-10-31 01:14:00.963'),(37,15,'n-in',87,118141,118228,'2026-10-31 01:14:30.963'),(38,15,'n-in',261,3146343,3146604,'2026-10-31 01:15:00.962'),(39,15,'n-in',87,131277,131364,'2026-10-31 01:19:04.793'),(40,15,'n-in',87,1048781,1048868,'2026-10-31 01:24:19.508'),(41,15,'n-in',87,1048781,1048868,'2026-10-31 01:24:21.509'),(42,15,'n-in',87,1048781,1048868,'2026-10-31 01:24:23.508'),(43,15,'n-in',174,2097562,2097736,'2026-10-31 01:24:31.508'),(44,25,'n-in',87,1048781,1048868,'2026-10-31 01:42:06.411'),(45,26,'n-in',9,14,23,'2026-10-31 01:42:34.411'),(46,27,'n-in',261,3146343,3146604,'2026-10-31 01:43:10.410'),(47,25,'n-in',87,1048781,2097606,'2026-10-31 01:44:08.410'),(48,15,'n-in',87,1048781,1048868,'2026-10-31 01:46:32.410'),(49,15,'n-in',474,6666,7140,'2026-10-31 11:55:41.028'),(50,15,'n-in',158,1111,1269,'2026-10-31 11:56:15.027'),(51,15,'n-in',158,1111,1269,'2026-10-31 11:56:35.028'),(52,15,'n-in',316,4444,4760,'2026-10-31 11:58:05.029'),(53,15,'n-in',237,3333,3570,'2026-10-31 11:58:45.028'),(54,30,'n-in',79,1111,1190,'2026-10-31 12:08:42.770'),(55,30,'n-in',9,9,18,'2026-10-31 12:08:42.770'),(56,30,'n-in',70,1102,1172,'2026-10-31 12:08:44.769'),(57,30,'n-in',9,9,18,'2026-10-31 12:08:44.769'),(58,30,'n-in',70,1102,1172,'2026-10-31 12:08:46.768'),(59,30,'n-in',9,9,18,'2026-10-31 12:08:46.768'),(60,30,'n-in',70,1102,1172,'2026-10-31 12:08:48.769'),(61,30,'n-in',9,9,18,'2026-10-31 12:08:48.769'),(62,15,'n-in',79,1111,1190,'2026-10-31 12:08:50.769'),(63,31,'n-in',79,1111,1190,'2026-10-31 12:14:37.786'),(64,15,'n-in',79,1111,1190,'2026-10-31 12:14:45.785'),(65,15,'n-in',79,1111,1190,'2026-10-31 12:28:04.750'),(66,15,'n-in',79,1111,1190,'2026-10-31 12:28:10.750'),(67,15,'n-in',79,1111,1190,'2026-10-31 12:28:36.751'),(68,15,'n-in',27,27,54,'2026-10-31 12:55:44.662'),(69,15,'n-in',16,8,24,'2026-10-31 12:56:04.661'),(70,15,'n-in',16,8,24,'2026-10-31 12:56:10.661'),(71,15,'n-in',6,6,12,'2026-10-31 13:03:23.171'),(72,15,'n-in',30,30,60,'2026-10-31 13:03:43.172');
/*!40000 ALTER TABLE `traffic_stats` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(64) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `role` varchar(32) NOT NULL DEFAULT 'admin',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `display_name` varchar(64) DEFAULT NULL,
  `role_id` bigint unsigned NOT NULL DEFAULT '0',
  `status` varchar(16) NOT NULL DEFAULT 'active',
  `last_login_at` datetime(3) DEFAULT NULL,
  `package_id` bigint unsigned DEFAULT NULL,
  `balance` decimal(14,2) NOT NULL DEFAULT '0.00',
  `package_period` varchar(16) NOT NULL DEFAULT '',
  `invite_code` varchar(16) DEFAULT NULL,
  `inviter_id` bigint unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_role_id` (`role_id`),
  KEY `idx_users_package_id` (`package_id`),
  KEY `idx_users_status` (`status`),
  KEY `idx_users_invite_code` (`invite_code`),
  KEY `idx_users_inviter_id` (`inviter_id`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES (1,'admin','$2a$10$q/kIGVc2p7Ir107HMfKPiOpehc48XbdLGeejJcgoyRCGts3ArPGJ2','super_admin','2026-10-30 22:23:45.637','2026-10-31 15:32:16.282','超级管理员',1,'active','2026-10-31 15:32:16.282',12,23.00,'monthly','69jgumpq',NULL),(6,'agent','$2a$10$VCdiX/k/J6S2e2rGY03Fve3hqIC.h8cwSDTnmP5MC9WBilrWqtTna','agent','2026-10-31 00:04:46.826','2026-10-31 16:05:35.837','Agent 节点账号',4,'active','2026-10-31 16:05:35.837',NULL,0.00,'',NULL,NULL),(12,'invitee1','$2a$10$unYiWWug4OVj32/hCtPMpuTv2ovTXnjR9PA30ilhRU06bbHbOBTNy','user','2026-10-31 15:09:50.719','2026-10-31 15:32:16.447','',3,'active','2026-10-31 15:32:16.447',12,400.00,'monthly','dgpn4hx2',1);
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Dumping routines for database 'aoe_tunnel'
--
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-10-31  8:15:08
