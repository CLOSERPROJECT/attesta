/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19  Distrib 10.11.16-MariaDB, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: appwrite
-- ------------------------------------------------------
-- Server version	10.11.16-MariaDB-ubu2204

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `_1__metadata`
--

DROP TABLE IF EXISTS `_1__metadata`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1__metadata` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `attributes` mediumtext DEFAULT NULL,
  `indexes` mediumtext DEFAULT NULL,
  `documentSecurity` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=31 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1__metadata`
--

LOCK TABLES `_1__metadata` WRITE;
/*!40000 ALTER TABLE `_1__metadata` DISABLE KEYS */;
INSERT INTO `_1__metadata` VALUES
(1,'audit','2026-04-02 09:38:02.976','2026-04-02 09:38:02.976','[\"create(\\\"any\\\")\"]','audit','[{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"event\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"resource\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"userAgent\",\"type\":\"string\",\"size\":65534,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"ip\",\"type\":\"string\",\"size\":45,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"location\",\"type\":\"string\",\"size\":45,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"time\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"data\",\"type\":\"string\",\"size\":16777216,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"]}]','[{\"$id\":\"index2\",\"type\":\"key\",\"attributes\":[\"event\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"index4\",\"type\":\"key\",\"attributes\":[\"userId\",\"event\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"index5\",\"type\":\"key\",\"attributes\":[\"resource\",\"event\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"index-time\",\"type\":\"key\",\"attributes\":[\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]}]',1),
(2,'databases','2026-04-02 09:38:03.090','2026-04-02 09:38:03.090','[\"create(\\\"any\\\")\"]','databases','[{\"$id\":\"name\",\"type\":\"string\",\"size\":256,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"enabled\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":false,\"default\":true,\"array\":false},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"originalId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"default\":null,\"array\":false},{\"$id\":\"type\",\"type\":\"string\",\"size\":128,\"required\":false,\"default\":\"tablesdb\",\"signed\":true,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_fulltext_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(3,'attributes','2026-04-02 09:38:03.171','2026-04-02 09:38:03.171','[\"create(\\\"any\\\")\"]','attributes','[{\"$id\":\"databaseInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"databaseId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":false,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"collectionInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"collectionId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"key\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"type\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"status\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"error\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"size\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"required\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"default\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"casting\"]},{\"$id\":\"signed\",\"type\":\"boolean\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"array\",\"type\":\"boolean\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"format\",\"type\":\"string\",\"size\":64,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"formatOptions\",\"type\":\"string\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":{},\"array\":false,\"filters\":[\"json\",\"range\",\"enum\"]},{\"$id\":\"filters\",\"type\":\"string\",\"size\":64,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"options\",\"type\":\"string\",\"size\":16384,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"json\"]}]','[{\"$id\":\"_key_db_collection\",\"type\":\"key\",\"attributes\":[\"databaseInternalId\",\"collectionInternalId\"],\"lengths\":[null,null],\"orders\":[\"ASC\",\"ASC\"]}]',1),
(4,'indexes','2026-04-02 09:38:03.254','2026-04-02 09:38:03.254','[\"create(\\\"any\\\")\"]','indexes','[{\"$id\":\"databaseInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"databaseId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":false,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"collectionInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"collectionId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"key\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"type\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"status\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"error\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"attributes\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"lengths\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"orders\",\"type\":\"string\",\"format\":\"\",\"size\":4,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]}]','[{\"$id\":\"_key_db_collection\",\"type\":\"key\",\"attributes\":[\"databaseInternalId\",\"collectionInternalId\"],\"lengths\":[null,null],\"orders\":[\"ASC\",\"ASC\"]}]',1),
(5,'functions','2026-04-02 09:38:03.413','2026-04-02 09:38:03.413','[\"create(\\\"any\\\")\"]','functions','[{\"$id\":\"execute\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"name\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"enabled\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"live\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"installationId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"installationInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerRepositoryId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"repositoryId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"repositoryInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerBranch\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerRootDirectory\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerSilentMode\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":false,\"default\":false,\"array\":false},{\"$id\":\"logging\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"runtime\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentCreatedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"latestDeploymentId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"latestDeploymentInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"latestDeploymentCreatedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"latestDeploymentStatus\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[]},{\"$id\":\"vars\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryVariables\"]},{\"$id\":\"varsProject\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryProjectVariables\"]},{\"$id\":\"events\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"scheduleInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"scheduleId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"schedule\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"timeout\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"version\",\"type\":\"string\",\"format\":\"\",\"size\":8,\"signed\":true,\"required\":false,\"default\":\"v5\",\"array\":false,\"filters\":[]},{\"array\":false,\"$id\":\"entrypoint\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"array\":false,\"$id\":\"commands\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"array\":false,\"$id\":\"specification\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":false,\"required\":false,\"default\":\"s-1vcpu-512mb\",\"filters\":[]},{\"$id\":\"scopes\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":[],\"array\":true,\"filters\":[]}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_enabled\",\"type\":\"key\",\"attributes\":[\"enabled\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_installationId\",\"type\":\"key\",\"attributes\":[\"installationId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_installationInternalId\",\"type\":\"key\",\"attributes\":[\"installationInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerRepositoryId\",\"type\":\"key\",\"attributes\":[\"providerRepositoryId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_repositoryId\",\"type\":\"key\",\"attributes\":[\"repositoryId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_repositoryInternalId\",\"type\":\"key\",\"attributes\":[\"repositoryInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_runtime\",\"type\":\"key\",\"attributes\":[\"runtime\"],\"lengths\":[64],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentId\",\"type\":\"key\",\"attributes\":[\"deploymentId\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(6,'sites','2026-04-02 09:38:03.561','2026-04-02 09:38:03.561','[\"create(\\\"any\\\")\"]','sites','[{\"$id\":\"name\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"enabled\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"live\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"installationId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"installationInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerRepositoryId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"repositoryId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"repositoryInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerBranch\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerRootDirectory\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerSilentMode\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":false,\"default\":false,\"array\":false},{\"$id\":\"logging\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"framework\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"array\":false,\"$id\":\"outputDirectory\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"array\":false,\"$id\":\"buildCommand\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"array\":false,\"$id\":\"installCommand\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"$id\":\"fallbackFile\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentCreatedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"deploymentScreenshotLight\",\"type\":\"string\",\"format\":\"\",\"size\":32,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentScreenshotDark\",\"type\":\"string\",\"format\":\"\",\"size\":32,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"latestDeploymentId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"latestDeploymentInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"latestDeploymentCreatedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"latestDeploymentStatus\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[]},{\"$id\":\"vars\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryVariables\"]},{\"$id\":\"varsProject\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryProjectVariables\"]},{\"$id\":\"timeout\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"array\":false,\"$id\":\"specification\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":false,\"required\":false,\"default\":\"s-1vcpu-512mb\",\"filters\":[]},{\"$id\":\"buildRuntime\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":true,\"default\":\"\",\"array\":false,\"filters\":[]},{\"$id\":\"adapter\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_enabled\",\"type\":\"key\",\"attributes\":[\"enabled\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_installationId\",\"type\":\"key\",\"attributes\":[\"installationId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_installationInternalId\",\"type\":\"key\",\"attributes\":[\"installationInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerRepositoryId\",\"type\":\"key\",\"attributes\":[\"providerRepositoryId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_repositoryId\",\"type\":\"key\",\"attributes\":[\"repositoryId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_repositoryInternalId\",\"type\":\"key\",\"attributes\":[\"repositoryInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_framework\",\"type\":\"key\",\"attributes\":[\"framework\"],\"lengths\":[64],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentId\",\"type\":\"key\",\"attributes\":[\"deploymentId\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(7,'deployments','2026-04-02 09:38:03.700','2026-04-02 09:38:03.700','[\"create(\\\"any\\\")\"]','deployments','[{\"$id\":\"resourceInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceType\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"array\":false,\"$id\":\"entrypoint\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"array\":false,\"$id\":\"buildCommands\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"array\":false,\"$id\":\"buildOutput\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"$id\":\"sourcePath\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"type\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"installationId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"installationInternalId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerRepositoryId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"repositoryId\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"repositoryInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerRepositoryName\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerRepositoryOwner\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerRepositoryUrl\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerCommitHash\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerCommitAuthorUrl\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerCommitAuthor\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerCommitMessage\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerCommitUrl\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerBranch\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerBranchUrl\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerRootDirectory\",\"type\":\"string\",\"signed\":true,\"size\":255,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"providerCommentId\",\"type\":\"string\",\"signed\":true,\"size\":2048,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"sourceSize\",\"type\":\"integer\",\"format\":\"\",\"size\":8,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"sourceMetadata\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"sourceChunksTotal\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"sourceChunksUploaded\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"activate\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":false,\"array\":false,\"filters\":[]},{\"$id\":\"screenshotLight\",\"type\":\"string\",\"format\":\"\",\"size\":32,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"screenshotDark\",\"type\":\"string\",\"format\":\"\",\"size\":32,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"buildStartedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"buildEndedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"buildDuration\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"buildSize\",\"type\":\"integer\",\"format\":\"\",\"size\":8,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"totalSize\",\"type\":\"integer\",\"format\":\"\",\"size\":8,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"status\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":\"waiting\",\"array\":false,\"filters\":[]},{\"$id\":\"buildPath\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[]},{\"$id\":\"buildLogs\",\"type\":\"string\",\"format\":\"\",\"size\":1000000,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[]},{\"$id\":\"adapter\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[]},{\"$id\":\"fallbackFile\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_resource\",\"type\":\"key\",\"attributes\":[\"resourceId\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_resource_type\",\"type\":\"key\",\"attributes\":[\"resourceType\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_sourceSize\",\"type\":\"key\",\"attributes\":[\"sourceSize\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_buildSize\",\"type\":\"key\",\"attributes\":[\"buildSize\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_totalSize\",\"type\":\"key\",\"attributes\":[\"totalSize\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_buildDuration\",\"type\":\"key\",\"attributes\":[\"buildDuration\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_activate\",\"type\":\"key\",\"attributes\":[\"activate\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_type\",\"type\":\"key\",\"attributes\":[\"type\"],\"lengths\":[32],\"orders\":[\"ASC\"]},{\"$id\":\"_key_status\",\"type\":\"key\",\"attributes\":[\"status\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_resourceId_resourceType\",\"type\":\"key\",\"attributes\":[\"resourceId\",\"resourceType\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_resource_internal_id\",\"type\":\"key\",\"attributes\":[\"resourceInternalId\"],\"lengths\":[],\"orders\":[]}]',1),
(8,'executions','2026-04-02 09:38:03.814','2026-04-02 09:38:03.814','[\"create(\\\"any\\\")\"]','executions','[{\"$id\":\"resourceInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceType\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deploymentId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"array\":false,\"$id\":\"trigger\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"$id\":\"status\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"duration\",\"type\":\"double\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"errors\",\"type\":\"string\",\"format\":\"\",\"size\":1000000,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"logs\",\"type\":\"string\",\"format\":\"\",\"size\":1000000,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"array\":false,\"$id\":\"requestMethod\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"array\":false,\"$id\":\"requestPath\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"filters\":[]},{\"$id\":\"requestHeaders\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"responseStatusCode\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"responseHeaders\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"scheduledAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"scheduleInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"scheduleId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_resource\",\"type\":\"key\",\"attributes\":[\"resourceInternalId\",\"resourceType\",\"resourceId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_trigger\",\"type\":\"key\",\"attributes\":[\"trigger\"],\"lengths\":[32],\"orders\":[\"ASC\"]},{\"$id\":\"_key_status\",\"type\":\"key\",\"attributes\":[\"status\"],\"lengths\":[32],\"orders\":[\"ASC\"]},{\"$id\":\"_key_requestMethod\",\"type\":\"key\",\"attributes\":[\"requestMethod\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_requestPath\",\"type\":\"key\",\"attributes\":[\"requestPath\"],\"lengths\":[255],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deployment\",\"type\":\"key\",\"attributes\":[\"deploymentId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_responseStatusCode\",\"type\":\"key\",\"attributes\":[\"responseStatusCode\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_duration\",\"type\":\"key\",\"attributes\":[\"duration\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_function_internal_id\",\"type\":\"key\",\"attributes\":[\"resourceInternalId\"],\"lengths\":[],\"orders\":[]}]',1),
(9,'variables','2026-04-02 09:38:03.955','2026-04-02 09:38:03.955','[\"create(\\\"any\\\")\"]','variables','[{\"$id\":\"resourceType\",\"type\":\"string\",\"format\":\"\",\"size\":100,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"key\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"value\",\"type\":\"string\",\"format\":\"\",\"size\":8192,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"secret\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":false,\"array\":false,\"filters\":[]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_resourceInternalId\",\"type\":\"key\",\"attributes\":[\"resourceInternalId\"],\"lengths\":[null],\"orders\":[]},{\"$id\":\"_key_resourceType\",\"type\":\"key\",\"attributes\":[\"resourceType\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_resourceId_resourceType\",\"type\":\"key\",\"attributes\":[\"resourceId\",\"resourceType\"],\"lengths\":[null,null],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_uniqueKey\",\"type\":\"unique\",\"attributes\":[\"resourceId\",\"key\",\"resourceType\"],\"lengths\":[null,null,null],\"orders\":[\"ASC\",\"ASC\",\"ASC\"]},{\"$id\":\"_key_key\",\"type\":\"key\",\"attributes\":[\"key\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_fulltext_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_resource_internal_id_resource_type\",\"type\":\"key\",\"attributes\":[\"resourceInternalId\",\"resourceType\"],\"lengths\":[],\"orders\":[]}]',1),
(10,'migrations','2026-04-02 09:38:04.084','2026-04-02 09:38:04.084','[\"create(\\\"any\\\")\"]','migrations','[{\"$id\":\"status\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"stage\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"source\",\"type\":\"string\",\"format\":\"\",\"size\":8192,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"destination\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"credentials\",\"type\":\"string\",\"format\":\"\",\"size\":65536,\"signed\":true,\"required\":false,\"default\":[],\"array\":false,\"filters\":[\"json\",\"encrypt\"]},{\"$id\":\"options\",\"type\":\"string\",\"format\":\"\",\"size\":65536,\"signed\":true,\"required\":false,\"default\":[],\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"resources\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":[],\"array\":true,\"filters\":[]},{\"$id\":\"statusCounters\",\"type\":\"string\",\"format\":\"\",\"size\":3000,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"resourceData\",\"type\":\"string\",\"format\":\"\",\"size\":131070,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"errors\",\"type\":\"string\",\"format\":\"\",\"size\":1000000,\"signed\":true,\"required\":true,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceType\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_status\",\"type\":\"key\",\"attributes\":[\"status\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_stage\",\"type\":\"key\",\"attributes\":[\"stage\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_source\",\"type\":\"key\",\"attributes\":[\"source\"],\"lengths\":[255],\"orders\":[\"ASC\"]},{\"$id\":\"_key_resource_id\",\"type\":\"key\",\"attributes\":[\"resourceId\"],\"lengths\":[null],\"orders\":[\"DESC\"]},{\"$id\":\"_fulltext_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]}]',1),
(11,'resourceTokens','2026-04-02 09:38:04.165','2026-04-02 09:38:04.165','[\"create(\\\"any\\\")\"]','resourceTokens','[{\"$id\":\"resourceId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceType\",\"type\":\"string\",\"format\":\"\",\"size\":100,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"secret\",\"type\":\"string\",\"format\":\"\",\"size\":512,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"expire\",\"type\":\"datetime\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]}]','[{\"$id\":\"_key_expiry_date\",\"type\":\"key\",\"attributes\":[\"expire\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_accessedAt\",\"type\":\"key\",\"attributes\":[\"accessedAt\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_resourceInternalId_resourceType\",\"type\":\"key\",\"attributes\":[\"resourceInternalId\",\"resourceType\"],\"lengths\":[],\"orders\":[]}]',1),
(12,'transactions','2026-04-02 09:38:04.248','2026-04-02 09:38:04.248','[\"create(\\\"any\\\")\"]','transactions','[{\"$id\":\"status\",\"type\":\"string\",\"size\":16,\"signed\":true,\"required\":false,\"default\":\"pending\",\"array\":false,\"filters\":[]},{\"$id\":\"operations\",\"type\":\"integer\",\"size\":0,\"signed\":false,\"required\":true,\"default\":0,\"array\":false,\"filters\":[]},{\"$id\":\"expiresAt\",\"type\":\"datetime\",\"size\":0,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]}]','[{\"$id\":\"_key_expiresAt\",\"type\":\"key\",\"attributes\":[\"expiresAt\"],\"lengths\":[],\"orders\":[\"DESC\"]}]',1),
(13,'transactionLogs','2026-04-02 09:38:04.323','2026-04-02 09:38:04.323','[\"create(\\\"any\\\")\"]','transactionLogs','[{\"$id\":\"transactionInternalId\",\"type\":\"string\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"databaseInternalId\",\"type\":\"string\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"collectionInternalId\",\"type\":\"string\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"documentId\",\"type\":\"string\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"action\",\"type\":\"string\",\"size\":32,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"data\",\"type\":\"string\",\"size\":5000000,\"signed\":false,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"json\"]}]','[{\"$id\":\"_key_transaction\",\"type\":\"key\",\"attributes\":[\"transactionInternalId\"],\"lengths\":[],\"orders\":[]}]',1),
(14,'cache','2026-04-02 09:38:04.410','2026-04-02 09:38:04.410','[\"create(\\\"any\\\")\"]','cache','[{\"$id\":\"resource\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"resourceType\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"mimeType\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"signature\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_accessedAt\",\"type\":\"key\",\"attributes\":[\"accessedAt\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_resource\",\"type\":\"key\",\"attributes\":[\"resource\"],\"lengths\":[],\"orders\":[]}]',1),
(15,'users','2026-04-02 09:38:04.586','2026-04-02 09:38:04.586','[\"create(\\\"any\\\")\"]','users','[{\"$id\":\"name\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"email\",\"type\":\"string\",\"format\":\"\",\"size\":320,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"phone\",\"type\":\"string\",\"format\":\"\",\"size\":16,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"status\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"labels\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"passwordHistory\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"password\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"hash\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":\"argon2\",\"array\":false,\"filters\":[]},{\"$id\":\"hashOptions\",\"type\":\"string\",\"format\":\"\",\"size\":65535,\"signed\":true,\"required\":false,\"default\":{\"type\":\"argon2\",\"memory_cost\":65536,\"time_cost\":4,\"threads\":3},\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"passwordUpdate\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"prefs\",\"type\":\"string\",\"format\":\"\",\"size\":65535,\"signed\":true,\"required\":false,\"default\":{},\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"registration\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"emailVerification\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"phoneVerification\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"reset\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"mfa\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"mfaRecoveryCodes\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":[],\"array\":true,\"filters\":[\"encrypt\"]},{\"$id\":\"authenticators\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryAuthenticators\"]},{\"$id\":\"sessions\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQuerySessions\"]},{\"$id\":\"tokens\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryTokens\"]},{\"$id\":\"challenges\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryChallenges\"]},{\"$id\":\"memberships\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryMemberships\"]},{\"$id\":\"targets\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryTargets\"]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"userSearch\"]},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"emailCanonical\",\"type\":\"string\",\"format\":\"\",\"size\":320,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"emailIsFree\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"emailIsDisposable\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"emailIsCorporate\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"emailIsCanonical\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_email\",\"type\":\"unique\",\"attributes\":[\"email\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_phone\",\"type\":\"unique\",\"attributes\":[\"phone\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_status\",\"type\":\"key\",\"attributes\":[\"status\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_passwordUpdate\",\"type\":\"key\",\"attributes\":[\"passwordUpdate\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_registration\",\"type\":\"key\",\"attributes\":[\"registration\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_emailVerification\",\"type\":\"key\",\"attributes\":[\"emailVerification\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_phoneVerification\",\"type\":\"key\",\"attributes\":[\"phoneVerification\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_accessedAt\",\"type\":\"key\",\"attributes\":[\"accessedAt\"],\"lengths\":[],\"orders\":[]}]',1),
(16,'tokens','2026-04-02 09:38:04.661','2026-04-02 09:38:04.661','[\"create(\\\"any\\\")\"]','tokens','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"type\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"secret\",\"type\":\"string\",\"format\":\"\",\"size\":512,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"expire\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"userAgent\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"ip\",\"type\":\"string\",\"format\":\"\",\"size\":45,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(17,'authenticators','2026-04-02 09:38:04.730','2026-04-02 09:38:04.730','[\"create(\\\"any\\\")\"]','authenticators','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"type\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"verified\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":false,\"array\":false,\"filters\":[]},{\"$id\":\"data\",\"type\":\"string\",\"format\":\"\",\"size\":65535,\"signed\":true,\"required\":false,\"default\":[],\"array\":false,\"filters\":[\"json\",\"encrypt\"]}]','[{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(18,'challenges','2026-04-02 09:38:04.800','2026-04-02 09:38:04.800','[\"create(\\\"any\\\")\"]','challenges','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"type\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"token\",\"type\":\"string\",\"format\":\"\",\"size\":512,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"code\",\"type\":\"string\",\"format\":\"\",\"size\":512,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"expire\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]}]','[{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(19,'sessions','2026-04-02 09:38:04.876','2026-04-02 09:38:04.876','[\"create(\\\"any\\\")\"]','sessions','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"provider\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerUid\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerAccessToken\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"providerAccessTokenExpiry\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"providerRefreshToken\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"secret\",\"type\":\"string\",\"format\":\"\",\"size\":512,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"userAgent\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"ip\",\"type\":\"string\",\"format\":\"\",\"size\":45,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"countryCode\",\"type\":\"string\",\"format\":\"\",\"size\":2,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"osCode\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"osName\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"osVersion\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"clientType\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"clientCode\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"clientName\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"clientVersion\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"clientEngine\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"clientEngineVersion\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deviceName\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deviceBrand\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deviceModel\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"factors\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":[],\"array\":true,\"filters\":[]},{\"$id\":\"expire\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"mfaUpdatedAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]}]','[{\"$id\":\"_key_provider_providerUid\",\"type\":\"key\",\"attributes\":[\"provider\",\"providerUid\"],\"lengths\":[null,128],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(20,'identities','2026-04-02 09:38:04.980','2026-04-02 09:38:04.980','[\"create(\\\"any\\\")\"]','identities','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"provider\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerUid\",\"type\":\"string\",\"format\":\"\",\"size\":2048,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerEmail\",\"type\":\"string\",\"format\":\"\",\"size\":320,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerAccessToken\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"providerAccessTokenExpiry\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"providerRefreshToken\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"secrets\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":[],\"array\":false,\"filters\":[\"json\",\"encrypt\"]},{\"$id\":\"scopes\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"expire\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"required\":false,\"signed\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]}]','[{\"$id\":\"_key_userInternalId_provider_providerUid\",\"type\":\"unique\",\"attributes\":[\"userInternalId\",\"provider\",\"providerUid\"],\"lengths\":[11,null,128],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_provider_providerUid\",\"type\":\"unique\",\"attributes\":[\"provider\",\"providerUid\"],\"lengths\":[null,128],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_provider\",\"type\":\"key\",\"attributes\":[\"provider\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerUid\",\"type\":\"key\",\"attributes\":[\"providerUid\"],\"lengths\":[255],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerEmail\",\"type\":\"key\",\"attributes\":[\"providerEmail\"],\"lengths\":[255],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerAccessTokenExpiry\",\"type\":\"key\",\"attributes\":[\"providerAccessTokenExpiry\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(21,'teams','2026-04-02 09:38:05.095','2026-04-02 09:38:05.095','[\"create(\\\"any\\\")\"]','teams','[{\"$id\":\"name\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"total\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"prefs\",\"type\":\"string\",\"format\":\"\",\"size\":65535,\"signed\":true,\"required\":false,\"default\":{},\"array\":false,\"filters\":[\"json\"]}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_total\",\"type\":\"key\",\"attributes\":[\"total\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(22,'memberships','2026-04-02 09:38:05.246','2026-04-02 09:38:05.246','[\"create(\\\"any\\\")\"]','memberships','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"teamInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"teamId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"roles\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"invited\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"joined\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"confirm\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"secret\",\"type\":\"string\",\"format\":\"\",\"size\":256,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"encrypt\"]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_unique\",\"type\":\"unique\",\"attributes\":[\"teamInternalId\",\"userInternalId\"],\"lengths\":[null,null],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_team\",\"type\":\"key\",\"attributes\":[\"teamInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_teamId\",\"type\":\"key\",\"attributes\":[\"teamId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_invited\",\"type\":\"key\",\"attributes\":[\"invited\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_joined\",\"type\":\"key\",\"attributes\":[\"joined\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_confirm\",\"type\":\"key\",\"attributes\":[\"confirm\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(23,'buckets','2026-04-02 09:38:05.400','2026-04-02 09:38:05.400','[\"create(\\\"any\\\")\"]','buckets','[{\"$id\":\"enabled\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"name\",\"type\":\"string\",\"signed\":true,\"size\":128,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"fileSecurity\",\"type\":\"boolean\",\"signed\":true,\"size\":1,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false},{\"$id\":\"maximumFileSize\",\"type\":\"integer\",\"signed\":false,\"size\":8,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"allowedFileExtensions\",\"type\":\"string\",\"signed\":true,\"size\":64,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":true},{\"$id\":\"compression\",\"type\":\"string\",\"signed\":true,\"size\":10,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"encryption\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"antivirus\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"array\":false},{\"$id\":\"transformations\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":false,\"array\":false,\"default\":true},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_fulltext_name\",\"type\":\"fulltext\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_enabled\",\"type\":\"key\",\"attributes\":[\"enabled\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_fileSecurity\",\"type\":\"key\",\"attributes\":[\"fileSecurity\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_maximumFileSize\",\"type\":\"key\",\"attributes\":[\"maximumFileSize\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_encryption\",\"type\":\"key\",\"attributes\":[\"encryption\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_antivirus\",\"type\":\"key\",\"attributes\":[\"antivirus\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(24,'stats','2026-04-02 09:38:05.479','2026-04-02 09:38:05.479','[\"create(\\\"any\\\")\"]','stats','[{\"$id\":\"metric\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"region\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"value\",\"type\":\"integer\",\"format\":\"\",\"size\":8,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"time\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"period\",\"type\":\"string\",\"format\":\"\",\"size\":4,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_time\",\"type\":\"key\",\"attributes\":[\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]},{\"$id\":\"_key_period_time\",\"type\":\"key\",\"attributes\":[\"period\",\"time\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_metric_period_time\",\"type\":\"unique\",\"attributes\":[\"metric\",\"period\",\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]}]',1),
(25,'providers','2026-04-02 09:38:05.617','2026-04-02 09:38:05.617','[\"create(\\\"any\\\")\"]','providers','[{\"$id\":\"name\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"provider\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"type\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"enabled\",\"type\":\"boolean\",\"signed\":true,\"size\":0,\"format\":\"\",\"filters\":[],\"required\":true,\"default\":true,\"array\":false},{\"$id\":\"credentials\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"json\",\"encrypt\"]},{\"$id\":\"options\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":[],\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":65535,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[\"providerSearch\"]}]','[{\"$id\":\"_key_provider\",\"type\":\"key\",\"attributes\":[\"provider\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_name\",\"type\":\"fulltext\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_type\",\"type\":\"key\",\"attributes\":[\"type\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_enabled_type\",\"type\":\"key\",\"attributes\":[\"enabled\",\"type\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]}]',1),
(26,'messages','2026-04-02 09:38:05.722','2026-04-02 09:38:05.722','[\"create(\\\"any\\\")\"]','messages','[{\"$id\":\"providerType\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"status\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":\"processing\",\"array\":false,\"filters\":[]},{\"$id\":\"data\",\"type\":\"string\",\"format\":\"\",\"size\":65535,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[\"json\"]},{\"$id\":\"topics\",\"type\":\"string\",\"format\":\"\",\"size\":21845,\"signed\":true,\"required\":false,\"default\":[],\"array\":true,\"filters\":[]},{\"$id\":\"users\",\"type\":\"string\",\"format\":\"\",\"size\":21845,\"signed\":true,\"required\":false,\"default\":[],\"array\":true,\"filters\":[]},{\"$id\":\"targets\",\"type\":\"string\",\"format\":\"\",\"size\":21845,\"signed\":true,\"required\":false,\"default\":[],\"array\":true,\"filters\":[]},{\"$id\":\"scheduledAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"scheduleInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"scheduleId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"deliveredAt\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":false,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"deliveryErrors\",\"type\":\"string\",\"format\":\"\",\"size\":65535,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"deliveredTotal\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":0,\"array\":false,\"filters\":[]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[\"messageSearch\"]}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]}]',1),
(27,'topics','2026-04-02 09:38:05.854','2026-04-02 09:38:05.854','[\"create(\\\"any\\\")\"]','topics','[{\"$id\":\"name\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"subscribe\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":false,\"default\":null,\"array\":true,\"filters\":[]},{\"$id\":\"emailTotal\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":0,\"array\":false,\"filters\":[]},{\"$id\":\"smsTotal\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":0,\"array\":false,\"filters\":[]},{\"$id\":\"pushTotal\",\"type\":\"integer\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":0,\"array\":false,\"filters\":[]},{\"$id\":\"targets\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[\"subQueryTopicTargets\"]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":\"\",\"array\":false,\"filters\":[\"topicSearch\"]}]','[{\"$id\":\"_key_name\",\"type\":\"fulltext\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(28,'subscribers','2026-04-02 09:38:06.023','2026-04-02 09:38:06.023','[\"create(\\\"any\\\")\"]','subscribers','[{\"$id\":\"targetId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"targetInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"topicId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"topicInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerType\",\"type\":\"string\",\"format\":\"\",\"size\":128,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"search\",\"type\":\"string\",\"format\":\"\",\"size\":16384,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_targetId\",\"type\":\"key\",\"attributes\":[\"targetId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_targetInternalId\",\"type\":\"key\",\"attributes\":[\"targetInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_topicId\",\"type\":\"key\",\"attributes\":[\"topicId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_topicInternalId\",\"type\":\"key\",\"attributes\":[\"topicInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_unique_target_topic\",\"type\":\"unique\",\"attributes\":[\"targetInternalId\",\"topicInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_fulltext_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]}]',1),
(29,'targets','2026-04-02 09:38:06.127','2026-04-02 09:38:06.127','[\"create(\\\"any\\\")\"]','targets','[{\"$id\":\"userId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"userInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"sessionId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"sessionInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerType\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"providerInternalId\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"identifier\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":true,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"name\",\"type\":\"string\",\"format\":\"\",\"size\":255,\"signed\":true,\"required\":false,\"default\":null,\"array\":false,\"filters\":[]},{\"$id\":\"expired\",\"type\":\"boolean\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"default\":false,\"array\":false,\"filters\":[]}]','[{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerId\",\"type\":\"key\",\"attributes\":[\"providerId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_providerInternalId\",\"type\":\"key\",\"attributes\":[\"providerInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_identifier\",\"type\":\"unique\",\"attributes\":[\"identifier\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_expired\",\"type\":\"key\",\"attributes\":[\"expired\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_session_internal_id\",\"type\":\"key\",\"attributes\":[\"sessionInternalId\"],\"lengths\":[],\"orders\":[]}]',1),
(30,'bucket_1','2026-04-02 09:39:05.098','2026-04-02 09:39:05.098','[]','bucket_1','[{\"$id\":\"bucketId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"bucketInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"path\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"signature\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"mimeType\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"metadata\",\"type\":\"string\",\"size\":75000,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":null,\"format\":\"\"},{\"$id\":\"sizeOriginal\",\"type\":\"integer\",\"size\":8,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"sizeActual\",\"type\":\"integer\",\"size\":8,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"algorithm\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"comment\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLVersion\",\"type\":\"string\",\"size\":64,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLCipher\",\"type\":\"string\",\"size\":64,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLTag\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLIV\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"chunksTotal\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"chunksUploaded\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"transformedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_bucket\",\"type\":\"key\",\"attributes\":[\"bucketId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_signature\",\"type\":\"key\",\"attributes\":[\"signature\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_mimeType\",\"type\":\"key\",\"attributes\":[\"mimeType\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_sizeOriginal\",\"type\":\"key\",\"attributes\":[\"sizeOriginal\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_chunksTotal\",\"type\":\"key\",\"attributes\":[\"chunksTotal\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_chunksUploaded\",\"type\":\"key\",\"attributes\":[\"chunksUploaded\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_transformedAt\",\"type\":\"key\",\"attributes\":[\"transformedAt\"],\"lengths\":[],\"orders\":[]}]',0);
/*!40000 ALTER TABLE `_1__metadata` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1__metadata_perms`
--

DROP TABLE IF EXISTS `_1__metadata_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1__metadata_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=30 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1__metadata_perms`
--

LOCK TABLES `_1__metadata_perms` WRITE;
/*!40000 ALTER TABLE `_1__metadata_perms` DISABLE KEYS */;
INSERT INTO `_1__metadata_perms` VALUES
(3,'create','any','attributes'),
(1,'create','any','audit'),
(17,'create','any','authenticators'),
(23,'create','any','buckets'),
(14,'create','any','cache'),
(18,'create','any','challenges'),
(2,'create','any','databases'),
(7,'create','any','deployments'),
(8,'create','any','executions'),
(5,'create','any','functions'),
(20,'create','any','identities'),
(4,'create','any','indexes'),
(22,'create','any','memberships'),
(26,'create','any','messages'),
(10,'create','any','migrations'),
(25,'create','any','providers'),
(11,'create','any','resourceTokens'),
(19,'create','any','sessions'),
(6,'create','any','sites'),
(24,'create','any','stats'),
(28,'create','any','subscribers'),
(29,'create','any','targets'),
(21,'create','any','teams'),
(16,'create','any','tokens'),
(27,'create','any','topics'),
(13,'create','any','transactionLogs'),
(12,'create','any','transactions'),
(15,'create','any','users'),
(9,'create','any','variables');
/*!40000 ALTER TABLE `_1__metadata_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_attributes`
--

DROP TABLE IF EXISTS `_1_attributes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_attributes` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `databaseInternalId` varchar(255) DEFAULT NULL,
  `databaseId` varchar(255) DEFAULT NULL,
  `collectionInternalId` varchar(255) DEFAULT NULL,
  `collectionId` varchar(255) DEFAULT NULL,
  `key` varchar(255) DEFAULT NULL,
  `type` varchar(256) DEFAULT NULL,
  `status` varchar(16) DEFAULT NULL,
  `error` varchar(2048) DEFAULT NULL,
  `size` int(11) DEFAULT NULL,
  `required` tinyint(1) DEFAULT NULL,
  `default` text DEFAULT NULL,
  `signed` tinyint(1) DEFAULT NULL,
  `array` tinyint(1) DEFAULT NULL,
  `format` varchar(64) DEFAULT NULL,
  `formatOptions` text DEFAULT NULL,
  `filters` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`filters`)),
  `options` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_db_collection` (`databaseInternalId`,`collectionInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_attributes`
--

LOCK TABLES `_1_attributes` WRITE;
/*!40000 ALTER TABLE `_1_attributes` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_attributes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_attributes_perms`
--

DROP TABLE IF EXISTS `_1_attributes_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_attributes_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_attributes_perms`
--

LOCK TABLES `_1_attributes_perms` WRITE;
/*!40000 ALTER TABLE `_1_attributes_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_attributes_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_audit`
--

DROP TABLE IF EXISTS `_1_audit`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_audit` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `event` varchar(255) DEFAULT NULL,
  `resource` varchar(255) DEFAULT NULL,
  `userAgent` text DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  `location` varchar(45) DEFAULT NULL,
  `time` datetime(3) DEFAULT NULL,
  `data` longtext DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `index2` (`event`),
  KEY `index4` (`userId`,`event`),
  KEY `index5` (`resource`,`event`),
  KEY `index-time` (`time` DESC),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=45 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_audit`
--

LOCK TABLES `_1_audit` WRITE;
/*!40000 ALTER TABLE `_1_audit` DISABLE KEYS */;
INSERT INTO `_1_audit` VALUES
(1,'69ce39391a8155a5dc04','2026-04-02 09:39:05.108','2026-04-02 09:39:05.108','[]','1','bucket.create','bucket/org-assets','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:39:05.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"admin\",\"data\":\"\"}'),
(2,'69ce394cd44a7911dbcf','2026-04-02 09:39:24.869','2026-04-02 09:39:24.869','[]','1','team.create','team/69ce394c003479cb2d9b','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:39:24.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"admin\",\"data\":\"\"}'),
(3,'69ce3962bc82cb8528be','2026-04-02 09:39:46.772','2026-04-02 09:39:46.772','[]','1','team.delete','team/69ce394c003479cb2d9b','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:39:46.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"admin\",\"data\":\"\"}'),
(4,'69ce3a122555bb985de7','2026-04-02 09:42:42.152','2026-04-02 09:42:42.152','[]',NULL,'user.create','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:42:42.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(5,'69ce3b2128bb5234b1e4','2026-04-02 09:47:13.166','2026-04-02 09:47:13.166','[]','1','session.create','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:13.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(6,'69ce3b2132c3b669529a','2026-04-02 09:47:13.207','2026-04-02 09:47:13.207','[]','1','team.create','team/organization-1','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:13.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(7,'69ce3b213c7d42303cbd','2026-04-02 09:47:13.247','2026-04-02 09:47:13.247','[]',NULL,'user.update','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:13.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(8,'69ce3b214196f6751d1c','2026-04-02 09:47:13.268','2026-04-02 09:47:13.268','[]','1','session.delete','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:13.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(9,'69ce3b30e79f1000e051','2026-04-02 09:47:28.948','2026-04-02 09:47:28.948','[]','1','session.create','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:28.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(10,'69ce3b310dbafe0cd45c','2026-04-02 09:47:29.056','2026-04-02 09:47:29.056','[]','1','membership.create','team/organization-1','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:29.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(11,'69ce3b3117356ea4f298','2026-04-02 09:47:29.095','2026-04-02 09:47:29.095','[]','1','session.delete','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:29.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(12,'69ce3b3a02072ef11888','2026-04-02 09:47:38.008','2026-04-02 09:47:38.008','[]','2','membership.update','team/organization-1','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:38.000','{\"userId\":\"69ce3b31064c3cca2338\",\"userName\":\"org1@admin.org\",\"userEmail\":\"org1@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(13,'69ce3b3a0c1c4433868e','2026-04-02 09:47:38.049','2026-04-02 09:47:38.049','[]',NULL,'user.update','user/69ce3b31064c3cca2338','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:38.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(14,'69ce3b3f0089b38c93f6','2026-04-02 09:47:43.002','2026-04-02 09:47:43.002','[]','2','user.update','user/69ce3b31064c3cca2338','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:47:42.000','{\"userId\":\"69ce3b31064c3cca2338\",\"userName\":\"org1@admin.org\",\"userEmail\":\"org1@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(15,'69ce3b5b7a167ab8c4af','2026-04-02 09:48:11.500','2026-04-02 09:48:11.500','[]','2','team.update','team/organization-1','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:11.000','{\"userId\":\"69ce3b31064c3cca2338\",\"userName\":\"org1@admin.org\",\"userEmail\":\"org1@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(16,'69ce3b627439b52c91b2','2026-04-02 09:48:18.476','2026-04-02 09:48:18.476','[]','2','team.update','team/organization-1','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:18.000','{\"userId\":\"69ce3b31064c3cca2338\",\"userName\":\"org1@admin.org\",\"userEmail\":\"org1@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(17,'69ce3b69931e312d27bb','2026-04-02 09:48:25.602','2026-04-02 09:48:25.602','[]','2','team.update','team/organization-1','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:25.000','{\"userId\":\"69ce3b31064c3cca2338\",\"userName\":\"org1@admin.org\",\"userEmail\":\"org1@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(18,'69ce3b75e8611212bf0c','2026-04-02 09:48:37.951','2026-04-02 09:48:37.951','[]',NULL,'user.update','user/69ce3b31064c3cca2338','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:37.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(19,'69ce3b79d681888876bb','2026-04-02 09:48:41.878','2026-04-02 09:48:41.878','[]','2','session.delete','user/69ce3b31064c3cca2338','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:41.000','{\"userId\":\"69ce3b31064c3cca2338\",\"userName\":\"org1@admin.org\",\"userEmail\":\"org1@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(20,'69ce3b8694a96d25f9ab','2026-04-02 09:48:54.608','2026-04-02 09:48:54.608','[]','1','session.create','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:54.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(21,'69ce3b869b48b12ccb5d','2026-04-02 09:48:54.636','2026-04-02 09:48:54.636','[]','1','team.create','team/organization-2','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:54.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(22,'69ce3b86a33c67f2f9fc','2026-04-02 09:48:54.668','2026-04-02 09:48:54.668','[]',NULL,'user.update','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:54.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(23,'69ce3b86a7e1a3b22b0a','2026-04-02 09:48:54.687','2026-04-02 09:48:54.687','[]','1','session.delete','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:48:54.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(24,'69ce3b8f4415c26aedfc','2026-04-02 09:49:03.278','2026-04-02 09:49:03.278','[]','1','session.create','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:03.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(25,'69ce3b8f5d00b9dd41e5','2026-04-02 09:49:03.380','2026-04-02 09:49:03.380','[]','1','membership.create','team/organization-2','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:03.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(26,'69ce3b8f656aec28394a','2026-04-02 09:49:03.415','2026-04-02 09:49:03.415','[]','1','session.delete','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:03.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(27,'69ce3b99b423d71b09ec','2026-04-02 09:49:13.737','2026-04-02 09:49:13.737','[]','3','membership.update','team/organization-2','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:13.000','{\"userId\":\"69ce3b8f57ee7c5a7955\",\"userName\":\"org2@admin.org\",\"userEmail\":\"org2@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(28,'69ce3b99bd8292013d59','2026-04-02 09:49:13.776','2026-04-02 09:49:13.776','[]',NULL,'user.update','user/69ce3b8f57ee7c5a7955','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:13.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(29,'69ce3b9dadd76f13b6f7','2026-04-02 09:49:17.712','2026-04-02 09:49:17.712','[]','3','user.update','user/69ce3b8f57ee7c5a7955','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:17.000','{\"userId\":\"69ce3b8f57ee7c5a7955\",\"userName\":\"org2@admin.org\",\"userEmail\":\"org2@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(30,'69ce3bacdeac17ea51ac','2026-04-02 09:49:32.912','2026-04-02 09:49:32.912','[]','3','team.update','team/organization-2','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:32.000','{\"userId\":\"69ce3b8f57ee7c5a7955\",\"userName\":\"org2@admin.org\",\"userEmail\":\"org2@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(31,'69ce3bb14d8711418e9b','2026-04-02 09:49:37.317','2026-04-02 09:49:37.317','[]',NULL,'user.update','user/69ce3b8f57ee7c5a7955','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:37.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(32,'69ce3bb92545cff0d081','2026-04-02 09:49:45.152','2026-04-02 09:49:45.152','[]','3','session.delete','user/69ce3b8f57ee7c5a7955','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:45.000','{\"userId\":\"69ce3b8f57ee7c5a7955\",\"userName\":\"org2@admin.org\",\"userEmail\":\"org2@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(33,'69ce3bc4ddcc8ddf018f','2026-04-02 09:49:56.908','2026-04-02 09:49:56.908','[]','1','session.create','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:56.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(34,'69ce3bc4e40b2a52898b','2026-04-02 09:49:56.934','2026-04-02 09:49:56.934','[]','1','team.create','team/organization-3','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:56.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(35,'69ce3bc4ec3ab31452f6','2026-04-02 09:49:56.967','2026-04-02 09:49:56.967','[]',NULL,'user.update','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:56.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(36,'69ce3bc4f0ef707f9f3d','2026-04-02 09:49:56.986','2026-04-02 09:49:56.986','[]','1','session.delete','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:49:56.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(37,'69ce3bcd61219e1d95ea','2026-04-02 09:50:05.397','2026-04-02 09:50:05.397','[]','1','session.create','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:50:05.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(38,'69ce3bcd7c85b7af51e1','2026-04-02 09:50:05.510','2026-04-02 09:50:05.510','[]','1','membership.create','team/organization-3','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:50:05.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(39,'69ce3bcd8717dd3ad73a','2026-04-02 09:50:05.553','2026-04-02 09:50:05.553','[]','1','session.delete','user/64e7705962f0dae3f86d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:50:05.000','{\"userId\":\"64e7705962f0dae3f86d\",\"userName\":\"Platform Admin\",\"userEmail\":\"platform_admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(40,'69ce3bd31d12738fc2ef','2026-04-02 09:50:11.119','2026-04-02 09:50:11.119','[]','4','membership.update','team/organization-3','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:50:11.000','{\"userId\":\"69ce3bcd771ae8884a4d\",\"userName\":\"org3@admin.org\",\"userEmail\":\"org3@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(41,'69ce3bd326318cc72fab','2026-04-02 09:50:11.156','2026-04-02 09:50:11.156','[]',NULL,'user.update','user/69ce3bcd771ae8884a4d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:50:11.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}'),
(42,'69ce3bd86430ff018946','2026-04-02 09:50:16.410','2026-04-02 09:50:16.410','[]','4','user.update','user/69ce3bcd771ae8884a4d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:50:16.000','{\"userId\":\"69ce3bcd771ae8884a4d\",\"userName\":\"org3@admin.org\",\"userEmail\":\"org3@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(43,'69ce3bee67874079381d','2026-04-02 09:50:38.424','2026-04-02 09:50:38.424','[]','4','team.update','team/organization-3','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:50:38.000','{\"userId\":\"69ce3bcd771ae8884a4d\",\"userName\":\"org3@admin.org\",\"userEmail\":\"org3@admin.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":\"\"}'),
(44,'69ce3c2a0f207f449471','2026-04-02 09:51:38.061','2026-04-02 09:51:38.061','[]',NULL,'user.update','user/69ce3bcd771ae8884a4d','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','','2026-04-02 09:51:38.000','{\"userId\":\"\",\"userName\":\"attesta auth key\",\"userEmail\":\"app.attesta@service.localhost\",\"userType\":\"app\",\"mode\":\"default\",\"data\":\"\"}');
/*!40000 ALTER TABLE `_1_audit` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_audit_perms`
--

DROP TABLE IF EXISTS `_1_audit_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_audit_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_audit_perms`
--

LOCK TABLES `_1_audit_perms` WRITE;
/*!40000 ALTER TABLE `_1_audit_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_audit_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_authenticators`
--

DROP TABLE IF EXISTS `_1_authenticators`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_authenticators` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `verified` tinyint(1) DEFAULT NULL,
  `data` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_authenticators`
--

LOCK TABLES `_1_authenticators` WRITE;
/*!40000 ALTER TABLE `_1_authenticators` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_authenticators` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_authenticators_perms`
--

DROP TABLE IF EXISTS `_1_authenticators_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_authenticators_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_authenticators_perms`
--

LOCK TABLES `_1_authenticators_perms` WRITE;
/*!40000 ALTER TABLE `_1_authenticators_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_authenticators_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_bucket_1`
--

DROP TABLE IF EXISTS `_1_bucket_1`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_bucket_1` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `bucketId` varchar(255) DEFAULT NULL,
  `bucketInternalId` varchar(255) DEFAULT NULL,
  `name` varchar(2048) DEFAULT NULL,
  `path` varchar(2048) DEFAULT NULL,
  `signature` varchar(2048) DEFAULT NULL,
  `mimeType` varchar(255) DEFAULT NULL,
  `metadata` mediumtext DEFAULT NULL,
  `sizeOriginal` bigint(20) unsigned DEFAULT NULL,
  `sizeActual` bigint(20) unsigned DEFAULT NULL,
  `algorithm` varchar(255) DEFAULT NULL,
  `comment` varchar(2048) DEFAULT NULL,
  `openSSLVersion` varchar(64) DEFAULT NULL,
  `openSSLCipher` varchar(64) DEFAULT NULL,
  `openSSLTag` varchar(2048) DEFAULT NULL,
  `openSSLIV` varchar(2048) DEFAULT NULL,
  `chunksTotal` int(10) unsigned DEFAULT NULL,
  `chunksUploaded` int(10) unsigned DEFAULT NULL,
  `transformedAt` datetime(3) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_bucket` (`bucketId`),
  KEY `_key_name` (`name`(256)),
  KEY `_key_signature` (`signature`(256)),
  KEY `_key_mimeType` (`mimeType`),
  KEY `_key_sizeOriginal` (`sizeOriginal`),
  KEY `_key_chunksTotal` (`chunksTotal`),
  KEY `_key_chunksUploaded` (`chunksUploaded`),
  KEY `_key_transformedAt` (`transformedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_bucket_1`
--

LOCK TABLES `_1_bucket_1` WRITE;
/*!40000 ALTER TABLE `_1_bucket_1` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_bucket_1` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_bucket_1_perms`
--

DROP TABLE IF EXISTS `_1_bucket_1_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_bucket_1_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_bucket_1_perms`
--

LOCK TABLES `_1_bucket_1_perms` WRITE;
/*!40000 ALTER TABLE `_1_bucket_1_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_bucket_1_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_buckets`
--

DROP TABLE IF EXISTS `_1_buckets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_buckets` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `fileSecurity` tinyint(1) DEFAULT NULL,
  `maximumFileSize` bigint(20) unsigned DEFAULT NULL,
  `allowedFileExtensions` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`allowedFileExtensions`)),
  `compression` varchar(10) DEFAULT NULL,
  `encryption` tinyint(1) DEFAULT NULL,
  `antivirus` tinyint(1) DEFAULT NULL,
  `transformations` tinyint(1) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_enabled` (`enabled`),
  KEY `_key_name` (`name`),
  KEY `_key_fileSecurity` (`fileSecurity`),
  KEY `_key_maximumFileSize` (`maximumFileSize`),
  KEY `_key_encryption` (`encryption`),
  KEY `_key_antivirus` (`antivirus`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_fulltext_name` (`name`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_buckets`
--

LOCK TABLES `_1_buckets` WRITE;
/*!40000 ALTER TABLE `_1_buckets` DISABLE KEYS */;
INSERT INTO `_1_buckets` VALUES
(1,'org-assets','2026-04-02 09:39:04.943','2026-04-02 09:39:04.943','[]',1,'org-assets',0,30000000,'[]','none',1,1,1,'org-assets org-assets');
/*!40000 ALTER TABLE `_1_buckets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_buckets_perms`
--

DROP TABLE IF EXISTS `_1_buckets_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_buckets_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_buckets_perms`
--

LOCK TABLES `_1_buckets_perms` WRITE;
/*!40000 ALTER TABLE `_1_buckets_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_buckets_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_cache`
--

DROP TABLE IF EXISTS `_1_cache`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_cache` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `resource` varchar(255) DEFAULT NULL,
  `resourceType` varchar(255) DEFAULT NULL,
  `mimeType` varchar(255) DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  `signature` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_accessedAt` (`accessedAt`),
  KEY `_key_resource` (`resource`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_cache`
--

LOCK TABLES `_1_cache` WRITE;
/*!40000 ALTER TABLE `_1_cache` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_cache` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_cache_perms`
--

DROP TABLE IF EXISTS `_1_cache_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_cache_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_cache_perms`
--

LOCK TABLES `_1_cache_perms` WRITE;
/*!40000 ALTER TABLE `_1_cache_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_cache_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_challenges`
--

DROP TABLE IF EXISTS `_1_challenges`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_challenges` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `token` varchar(512) DEFAULT NULL,
  `code` varchar(512) DEFAULT NULL,
  `expire` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_user` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_challenges`
--

LOCK TABLES `_1_challenges` WRITE;
/*!40000 ALTER TABLE `_1_challenges` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_challenges` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_challenges_perms`
--

DROP TABLE IF EXISTS `_1_challenges_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_challenges_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_challenges_perms`
--

LOCK TABLES `_1_challenges_perms` WRITE;
/*!40000 ALTER TABLE `_1_challenges_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_challenges_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_databases`
--

DROP TABLE IF EXISTS `_1_databases`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_databases` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `search` text DEFAULT NULL,
  `originalId` varchar(255) DEFAULT NULL,
  `type` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_name` (`name`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_fulltext_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_databases`
--

LOCK TABLES `_1_databases` WRITE;
/*!40000 ALTER TABLE `_1_databases` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_databases` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_databases_perms`
--

DROP TABLE IF EXISTS `_1_databases_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_databases_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_databases_perms`
--

LOCK TABLES `_1_databases_perms` WRITE;
/*!40000 ALTER TABLE `_1_databases_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_databases_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_deployments`
--

DROP TABLE IF EXISTS `_1_deployments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_deployments` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `resourceInternalId` varchar(255) DEFAULT NULL,
  `resourceId` varchar(255) DEFAULT NULL,
  `resourceType` varchar(255) DEFAULT NULL,
  `entrypoint` varchar(2048) DEFAULT NULL,
  `buildCommands` text DEFAULT NULL,
  `buildOutput` text DEFAULT NULL,
  `sourcePath` text DEFAULT NULL,
  `type` varchar(2048) DEFAULT NULL,
  `installationId` varchar(255) DEFAULT NULL,
  `installationInternalId` varchar(255) DEFAULT NULL,
  `providerRepositoryId` varchar(255) DEFAULT NULL,
  `repositoryId` varchar(255) DEFAULT NULL,
  `repositoryInternalId` varchar(255) DEFAULT NULL,
  `providerRepositoryName` varchar(255) DEFAULT NULL,
  `providerRepositoryOwner` varchar(255) DEFAULT NULL,
  `providerRepositoryUrl` varchar(255) DEFAULT NULL,
  `providerCommitHash` varchar(255) DEFAULT NULL,
  `providerCommitAuthorUrl` varchar(255) DEFAULT NULL,
  `providerCommitAuthor` varchar(255) DEFAULT NULL,
  `providerCommitMessage` varchar(255) DEFAULT NULL,
  `providerCommitUrl` varchar(255) DEFAULT NULL,
  `providerBranch` varchar(255) DEFAULT NULL,
  `providerBranchUrl` varchar(255) DEFAULT NULL,
  `providerRootDirectory` varchar(255) DEFAULT NULL,
  `providerCommentId` varchar(2048) DEFAULT NULL,
  `sourceSize` bigint(20) unsigned DEFAULT NULL,
  `sourceMetadata` text DEFAULT NULL,
  `sourceChunksTotal` int(10) unsigned DEFAULT NULL,
  `sourceChunksUploaded` int(10) unsigned DEFAULT NULL,
  `activate` tinyint(1) DEFAULT NULL,
  `screenshotLight` varchar(32) DEFAULT NULL,
  `screenshotDark` varchar(32) DEFAULT NULL,
  `buildStartedAt` datetime(3) DEFAULT NULL,
  `buildEndedAt` datetime(3) DEFAULT NULL,
  `buildDuration` int(10) unsigned DEFAULT NULL,
  `buildSize` bigint(20) unsigned DEFAULT NULL,
  `totalSize` bigint(20) unsigned DEFAULT NULL,
  `status` varchar(16) DEFAULT NULL,
  `buildPath` text DEFAULT NULL,
  `buildLogs` mediumtext DEFAULT NULL,
  `adapter` varchar(16) DEFAULT NULL,
  `fallbackFile` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_resource` (`resourceId`),
  KEY `_key_resource_type` (`resourceType`),
  KEY `_key_sourceSize` (`sourceSize`),
  KEY `_key_buildSize` (`buildSize`),
  KEY `_key_totalSize` (`totalSize`),
  KEY `_key_buildDuration` (`buildDuration`),
  KEY `_key_activate` (`activate`),
  KEY `_key_type` (`type`(32)),
  KEY `_key_status` (`status`),
  KEY `_key_resourceId_resourceType` (`resourceId`,`resourceType`),
  KEY `_key_resource_internal_id` (`resourceInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_deployments`
--

LOCK TABLES `_1_deployments` WRITE;
/*!40000 ALTER TABLE `_1_deployments` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_deployments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_deployments_perms`
--

DROP TABLE IF EXISTS `_1_deployments_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_deployments_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_deployments_perms`
--

LOCK TABLES `_1_deployments_perms` WRITE;
/*!40000 ALTER TABLE `_1_deployments_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_deployments_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_executions`
--

DROP TABLE IF EXISTS `_1_executions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_executions` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `resourceInternalId` varchar(255) DEFAULT NULL,
  `resourceId` varchar(255) DEFAULT NULL,
  `resourceType` varchar(255) DEFAULT NULL,
  `deploymentInternalId` varchar(255) DEFAULT NULL,
  `deploymentId` varchar(255) DEFAULT NULL,
  `trigger` varchar(128) DEFAULT NULL,
  `status` varchar(128) DEFAULT NULL,
  `duration` double DEFAULT NULL,
  `errors` mediumtext DEFAULT NULL,
  `logs` mediumtext DEFAULT NULL,
  `requestMethod` varchar(128) DEFAULT NULL,
  `requestPath` varchar(2048) DEFAULT NULL,
  `requestHeaders` text DEFAULT NULL,
  `responseStatusCode` int(11) DEFAULT NULL,
  `responseHeaders` text DEFAULT NULL,
  `scheduledAt` datetime(3) DEFAULT NULL,
  `scheduleInternalId` varchar(255) DEFAULT NULL,
  `scheduleId` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_resource` (`resourceInternalId`,`resourceType`,`resourceId`),
  KEY `_key_trigger` (`trigger`(32)),
  KEY `_key_status` (`status`(32)),
  KEY `_key_requestMethod` (`requestMethod`),
  KEY `_key_requestPath` (`requestPath`(255)),
  KEY `_key_deployment` (`deploymentId`),
  KEY `_key_responseStatusCode` (`responseStatusCode`),
  KEY `_key_duration` (`duration`),
  KEY `_key_function_internal_id` (`resourceInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_executions`
--

LOCK TABLES `_1_executions` WRITE;
/*!40000 ALTER TABLE `_1_executions` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_executions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_executions_perms`
--

DROP TABLE IF EXISTS `_1_executions_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_executions_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_executions_perms`
--

LOCK TABLES `_1_executions_perms` WRITE;
/*!40000 ALTER TABLE `_1_executions_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_executions_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_functions`
--

DROP TABLE IF EXISTS `_1_functions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_functions` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `execute` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`execute`)),
  `name` varchar(2048) DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `live` tinyint(1) DEFAULT NULL,
  `installationId` varchar(255) DEFAULT NULL,
  `installationInternalId` varchar(255) DEFAULT NULL,
  `providerRepositoryId` varchar(255) DEFAULT NULL,
  `repositoryId` varchar(255) DEFAULT NULL,
  `repositoryInternalId` varchar(255) DEFAULT NULL,
  `providerBranch` varchar(255) DEFAULT NULL,
  `providerRootDirectory` varchar(255) DEFAULT NULL,
  `providerSilentMode` tinyint(1) DEFAULT NULL,
  `logging` tinyint(1) DEFAULT NULL,
  `runtime` varchar(2048) DEFAULT NULL,
  `deploymentInternalId` varchar(255) DEFAULT NULL,
  `deploymentId` varchar(255) DEFAULT NULL,
  `deploymentCreatedAt` datetime(3) DEFAULT NULL,
  `latestDeploymentId` varchar(255) DEFAULT NULL,
  `latestDeploymentInternalId` varchar(255) DEFAULT NULL,
  `latestDeploymentCreatedAt` datetime(3) DEFAULT NULL,
  `latestDeploymentStatus` varchar(16) DEFAULT NULL,
  `vars` text DEFAULT NULL,
  `varsProject` text DEFAULT NULL,
  `events` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`events`)),
  `scheduleInternalId` varchar(255) DEFAULT NULL,
  `scheduleId` varchar(255) DEFAULT NULL,
  `schedule` varchar(128) DEFAULT NULL,
  `timeout` int(11) DEFAULT NULL,
  `search` text DEFAULT NULL,
  `version` varchar(8) DEFAULT NULL,
  `entrypoint` text DEFAULT NULL,
  `commands` text DEFAULT NULL,
  `specification` varchar(128) DEFAULT NULL,
  `scopes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`scopes`)),
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_name` (`name`(256)),
  KEY `_key_enabled` (`enabled`),
  KEY `_key_installationId` (`installationId`),
  KEY `_key_installationInternalId` (`installationInternalId`),
  KEY `_key_providerRepositoryId` (`providerRepositoryId`),
  KEY `_key_repositoryId` (`repositoryId`),
  KEY `_key_repositoryInternalId` (`repositoryInternalId`),
  KEY `_key_runtime` (`runtime`(64)),
  KEY `_key_deploymentId` (`deploymentId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_functions`
--

LOCK TABLES `_1_functions` WRITE;
/*!40000 ALTER TABLE `_1_functions` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_functions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_functions_perms`
--

DROP TABLE IF EXISTS `_1_functions_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_functions_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_functions_perms`
--

LOCK TABLES `_1_functions_perms` WRITE;
/*!40000 ALTER TABLE `_1_functions_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_functions_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_identities`
--

DROP TABLE IF EXISTS `_1_identities`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_identities` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `provider` varchar(128) DEFAULT NULL,
  `providerUid` varchar(2048) DEFAULT NULL,
  `providerEmail` varchar(320) DEFAULT NULL,
  `providerAccessToken` text DEFAULT NULL,
  `providerAccessTokenExpiry` datetime(3) DEFAULT NULL,
  `providerRefreshToken` text DEFAULT NULL,
  `secrets` text DEFAULT NULL,
  `scopes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`scopes`)),
  `expire` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_userInternalId_provider_providerUid` (`userInternalId`(11),`provider`,`providerUid`(128)),
  UNIQUE KEY `_key_provider_providerUid` (`provider`,`providerUid`(128)),
  KEY `_key_userId` (`userId`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_key_provider` (`provider`),
  KEY `_key_providerUid` (`providerUid`(255)),
  KEY `_key_providerEmail` (`providerEmail`(255)),
  KEY `_key_providerAccessTokenExpiry` (`providerAccessTokenExpiry`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_identities`
--

LOCK TABLES `_1_identities` WRITE;
/*!40000 ALTER TABLE `_1_identities` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_identities` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_identities_perms`
--

DROP TABLE IF EXISTS `_1_identities_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_identities_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_identities_perms`
--

LOCK TABLES `_1_identities_perms` WRITE;
/*!40000 ALTER TABLE `_1_identities_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_identities_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_indexes`
--

DROP TABLE IF EXISTS `_1_indexes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_indexes` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `databaseInternalId` varchar(255) DEFAULT NULL,
  `databaseId` varchar(255) DEFAULT NULL,
  `collectionInternalId` varchar(255) DEFAULT NULL,
  `collectionId` varchar(255) DEFAULT NULL,
  `key` varchar(255) DEFAULT NULL,
  `type` varchar(16) DEFAULT NULL,
  `status` varchar(16) DEFAULT NULL,
  `error` varchar(2048) DEFAULT NULL,
  `attributes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`attributes`)),
  `lengths` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`lengths`)),
  `orders` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`orders`)),
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_db_collection` (`databaseInternalId`,`collectionInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_indexes`
--

LOCK TABLES `_1_indexes` WRITE;
/*!40000 ALTER TABLE `_1_indexes` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_indexes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_indexes_perms`
--

DROP TABLE IF EXISTS `_1_indexes_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_indexes_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_indexes_perms`
--

LOCK TABLES `_1_indexes_perms` WRITE;
/*!40000 ALTER TABLE `_1_indexes_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_indexes_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_memberships`
--

DROP TABLE IF EXISTS `_1_memberships`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_memberships` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `teamInternalId` varchar(255) DEFAULT NULL,
  `teamId` varchar(255) DEFAULT NULL,
  `roles` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`roles`)),
  `invited` datetime(3) DEFAULT NULL,
  `joined` datetime(3) DEFAULT NULL,
  `confirm` tinyint(1) DEFAULT NULL,
  `secret` varchar(256) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_unique` (`teamInternalId`,`userInternalId`),
  KEY `_key_user` (`userInternalId`),
  KEY `_key_team` (`teamInternalId`),
  KEY `_key_userId` (`userId`),
  KEY `_key_teamId` (`teamId`),
  KEY `_key_invited` (`invited`),
  KEY `_key_joined` (`joined`),
  KEY `_key_confirm` (`confirm`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_memberships`
--

LOCK TABLES `_1_memberships` WRITE;
/*!40000 ALTER TABLE `_1_memberships` DISABLE KEYS */;
INSERT INTO `_1_memberships` VALUES
(1,'69ce3b212fa2edcd785d','2026-04-02 09:47:13.195','2026-04-02 09:47:13.195','[\"read(\\\"user:64e7705962f0dae3f86d\\\")\",\"read(\\\"team:organization-1\\\")\",\"update(\\\"user:64e7705962f0dae3f86d\\\")\",\"update(\\\"team:organization-1\\/owner\\\")\",\"delete(\\\"user:64e7705962f0dae3f86d\\\")\",\"delete(\\\"team:organization-1\\/owner\\\")\"]','1','64e7705962f0dae3f86d','2','organization-1','[\"owner\"]','2026-04-02 09:47:13.195','2026-04-02 09:47:13.195',1,'{\"data\":\"\",\"method\":\"aes-128-gcm\",\"iv\":\"95bd57089044d119aa995b48\",\"tag\":\"f211344e98c8a5f16648c57c9c5694ac\",\"version\":\"1\"}','69ce3b212fa2edcd785d 64e7705962f0dae3f86d'),
(2,'69ce3b3108d806b9f17c','2026-04-02 09:47:29.036','2026-04-02 09:47:37.994','[\"read(\\\"any\\\")\",\"update(\\\"user:69ce3b31064c3cca2338\\\")\",\"update(\\\"team:organization-1\\/owner\\\")\",\"delete(\\\"user:69ce3b31064c3cca2338\\\")\",\"delete(\\\"team:organization-1\\/owner\\\")\"]','2','69ce3b31064c3cca2338','2','organization-1','[\"owner\"]','2026-04-02 09:47:29.036','2026-04-02 09:47:37.970',1,'{\"data\":\"ZfyagZGnTEleVNfEIJ+KUy\\/BluB2l7P5MKKJoEYNLdvjPWPGEHtmZIzIu9fKijdq8X7Bbr0MLi\\/cWMJ9vX7Y4g==\",\"method\":\"aes-128-gcm\",\"iv\":\"0824b44e747e06899720cc31\",\"tag\":\"d0f1439d5cc646a57c2cffd723b82f0c\",\"version\":\"1\"}','69ce3b3108d806b9f17c 69ce3b31064c3cca2338'),
(3,'69ce3b8696561c65b6fc','2026-04-02 09:48:54.616','2026-04-02 09:48:54.616','[\"read(\\\"user:64e7705962f0dae3f86d\\\")\",\"read(\\\"team:organization-2\\\")\",\"update(\\\"user:64e7705962f0dae3f86d\\\")\",\"update(\\\"team:organization-2\\/owner\\\")\",\"delete(\\\"user:64e7705962f0dae3f86d\\\")\",\"delete(\\\"team:organization-2\\/owner\\\")\"]','1','64e7705962f0dae3f86d','3','organization-2','[\"owner\"]','2026-04-02 09:48:54.615','2026-04-02 09:48:54.615',1,'{\"data\":\"\",\"method\":\"aes-128-gcm\",\"iv\":\"a16f35817d4df739a96f627d\",\"tag\":\"c986ab33e626f9e6db9d90e8b7812846\",\"version\":\"1\"}','69ce3b8696561c65b6fc 64e7705962f0dae3f86d'),
(4,'69ce3b8f5a73a7153cb8','2026-04-02 09:49:03.370','2026-04-02 09:49:13.721','[\"read(\\\"any\\\")\",\"update(\\\"user:69ce3b8f57ee7c5a7955\\\")\",\"update(\\\"team:organization-2\\/owner\\\")\",\"delete(\\\"user:69ce3b8f57ee7c5a7955\\\")\",\"delete(\\\"team:organization-2\\/owner\\\")\"]','3','69ce3b8f57ee7c5a7955','3','organization-2','[\"owner\"]','2026-04-02 09:49:03.370','2026-04-02 09:49:13.704',1,'{\"data\":\"8whiL4PptCzdHh0lA66fW4d+vM1NLtgxjoBNrNkxqg4\\/tZWSDigO1paEgA08PMb0esc6+ebnwgJM0raZdT138A==\",\"method\":\"aes-128-gcm\",\"iv\":\"ab93844b361d35f7103d60e6\",\"tag\":\"f397cb6e2b5119ee5af048490e76fb35\",\"version\":\"1\"}','69ce3b8f5a73a7153cb8 69ce3b8f57ee7c5a7955'),
(5,'69ce3bc4e172cfc13d70','2026-04-02 09:49:56.923','2026-04-02 09:49:56.923','[\"read(\\\"user:64e7705962f0dae3f86d\\\")\",\"read(\\\"team:organization-3\\\")\",\"update(\\\"user:64e7705962f0dae3f86d\\\")\",\"update(\\\"team:organization-3\\/owner\\\")\",\"delete(\\\"user:64e7705962f0dae3f86d\\\")\",\"delete(\\\"team:organization-3\\/owner\\\")\"]','1','64e7705962f0dae3f86d','4','organization-3','[\"owner\"]','2026-04-02 09:49:56.923','2026-04-02 09:49:56.923',1,'{\"data\":\"\",\"method\":\"aes-128-gcm\",\"iv\":\"c5a3daf547e9d66f91236af8\",\"tag\":\"bc1e5641478b81e70460c165a7336962\",\"version\":\"1\"}','69ce3bc4e172cfc13d70 64e7705962f0dae3f86d'),
(6,'69ce3bcd7a868f7df739','2026-04-02 09:50:05.502','2026-04-02 09:50:11.102','[\"read(\\\"any\\\")\",\"update(\\\"user:69ce3bcd771ae8884a4d\\\")\",\"update(\\\"team:organization-3\\/owner\\\")\",\"delete(\\\"user:69ce3bcd771ae8884a4d\\\")\",\"delete(\\\"team:organization-3\\/owner\\\")\"]','4','69ce3bcd771ae8884a4d','4','organization-3','[\"owner\"]','2026-04-02 09:50:05.501','2026-04-02 09:50:11.073',1,'{\"data\":\"rB0fq27nuk2gIEGdQKhF4JaB\\/COr\\/T9jqVTxxduzLWEGT0GOnefyqqpyzzitroX0zFe9Eip8+y4l3HpDNNfLDA==\",\"method\":\"aes-128-gcm\",\"iv\":\"ea29e72204507000092f8654\",\"tag\":\"8cce610bd806a27ea0e78af717d1374f\",\"version\":\"1\"}','69ce3bcd7a868f7df739 69ce3bcd771ae8884a4d');
/*!40000 ALTER TABLE `_1_memberships` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_memberships_perms`
--

DROP TABLE IF EXISTS `_1_memberships_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_memberships_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=34 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_memberships_perms`
--

LOCK TABLES `_1_memberships_perms` WRITE;
/*!40000 ALTER TABLE `_1_memberships_perms` DISABLE KEYS */;
INSERT INTO `_1_memberships_perms` VALUES
(6,'delete','team:organization-1/owner','69ce3b212fa2edcd785d'),
(5,'delete','user:64e7705962f0dae3f86d','69ce3b212fa2edcd785d'),
(2,'read','team:organization-1','69ce3b212fa2edcd785d'),
(1,'read','user:64e7705962f0dae3f86d','69ce3b212fa2edcd785d'),
(4,'update','team:organization-1/owner','69ce3b212fa2edcd785d'),
(3,'update','user:64e7705962f0dae3f86d','69ce3b212fa2edcd785d'),
(11,'delete','team:organization-1/owner','69ce3b3108d806b9f17c'),
(10,'delete','user:69ce3b31064c3cca2338','69ce3b3108d806b9f17c'),
(7,'read','any','69ce3b3108d806b9f17c'),
(9,'update','team:organization-1/owner','69ce3b3108d806b9f17c'),
(8,'update','user:69ce3b31064c3cca2338','69ce3b3108d806b9f17c'),
(17,'delete','team:organization-2/owner','69ce3b8696561c65b6fc'),
(16,'delete','user:64e7705962f0dae3f86d','69ce3b8696561c65b6fc'),
(13,'read','team:organization-2','69ce3b8696561c65b6fc'),
(12,'read','user:64e7705962f0dae3f86d','69ce3b8696561c65b6fc'),
(15,'update','team:organization-2/owner','69ce3b8696561c65b6fc'),
(14,'update','user:64e7705962f0dae3f86d','69ce3b8696561c65b6fc'),
(22,'delete','team:organization-2/owner','69ce3b8f5a73a7153cb8'),
(21,'delete','user:69ce3b8f57ee7c5a7955','69ce3b8f5a73a7153cb8'),
(18,'read','any','69ce3b8f5a73a7153cb8'),
(20,'update','team:organization-2/owner','69ce3b8f5a73a7153cb8'),
(19,'update','user:69ce3b8f57ee7c5a7955','69ce3b8f5a73a7153cb8'),
(28,'delete','team:organization-3/owner','69ce3bc4e172cfc13d70'),
(27,'delete','user:64e7705962f0dae3f86d','69ce3bc4e172cfc13d70'),
(24,'read','team:organization-3','69ce3bc4e172cfc13d70'),
(23,'read','user:64e7705962f0dae3f86d','69ce3bc4e172cfc13d70'),
(26,'update','team:organization-3/owner','69ce3bc4e172cfc13d70'),
(25,'update','user:64e7705962f0dae3f86d','69ce3bc4e172cfc13d70'),
(33,'delete','team:organization-3/owner','69ce3bcd7a868f7df739'),
(32,'delete','user:69ce3bcd771ae8884a4d','69ce3bcd7a868f7df739'),
(29,'read','any','69ce3bcd7a868f7df739'),
(31,'update','team:organization-3/owner','69ce3bcd7a868f7df739'),
(30,'update','user:69ce3bcd771ae8884a4d','69ce3bcd7a868f7df739');
/*!40000 ALTER TABLE `_1_memberships_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_messages`
--

DROP TABLE IF EXISTS `_1_messages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_messages` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `providerType` varchar(255) DEFAULT NULL,
  `status` varchar(255) DEFAULT NULL,
  `data` text DEFAULT NULL,
  `topics` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`topics`)),
  `users` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`users`)),
  `targets` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`targets`)),
  `scheduledAt` datetime(3) DEFAULT NULL,
  `scheduleInternalId` varchar(255) DEFAULT NULL,
  `scheduleId` varchar(255) DEFAULT NULL,
  `deliveredAt` datetime(3) DEFAULT NULL,
  `deliveryErrors` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`deliveryErrors`)),
  `deliveredTotal` int(11) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_messages`
--

LOCK TABLES `_1_messages` WRITE;
/*!40000 ALTER TABLE `_1_messages` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_messages` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_messages_perms`
--

DROP TABLE IF EXISTS `_1_messages_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_messages_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_messages_perms`
--

LOCK TABLES `_1_messages_perms` WRITE;
/*!40000 ALTER TABLE `_1_messages_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_messages_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_migrations`
--

DROP TABLE IF EXISTS `_1_migrations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_migrations` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `status` varchar(255) DEFAULT NULL,
  `stage` varchar(255) DEFAULT NULL,
  `source` varchar(8192) DEFAULT NULL,
  `destination` varchar(255) DEFAULT NULL,
  `credentials` mediumtext DEFAULT NULL,
  `options` mediumtext DEFAULT NULL,
  `resources` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`resources`)),
  `statusCounters` varchar(3000) DEFAULT NULL,
  `resourceData` mediumtext DEFAULT NULL,
  `errors` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`errors`)),
  `search` text DEFAULT NULL,
  `resourceId` varchar(255) DEFAULT NULL,
  `resourceType` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_status` (`status`),
  KEY `_key_stage` (`stage`),
  KEY `_key_source` (`source`(255)),
  KEY `_key_resource_id` (`resourceId` DESC),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_fulltext_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_migrations`
--

LOCK TABLES `_1_migrations` WRITE;
/*!40000 ALTER TABLE `_1_migrations` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_migrations` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_migrations_perms`
--

DROP TABLE IF EXISTS `_1_migrations_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_migrations_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_migrations_perms`
--

LOCK TABLES `_1_migrations_perms` WRITE;
/*!40000 ALTER TABLE `_1_migrations_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_migrations_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_providers`
--

DROP TABLE IF EXISTS `_1_providers`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_providers` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `provider` varchar(255) DEFAULT NULL,
  `type` varchar(128) DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `credentials` text DEFAULT NULL,
  `options` text DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_provider` (`provider`),
  KEY `_key_type` (`type`),
  KEY `_key_enabled_type` (`enabled`,`type`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_name` (`name`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_providers`
--

LOCK TABLES `_1_providers` WRITE;
/*!40000 ALTER TABLE `_1_providers` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_providers` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_providers_perms`
--

DROP TABLE IF EXISTS `_1_providers_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_providers_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_providers_perms`
--

LOCK TABLES `_1_providers_perms` WRITE;
/*!40000 ALTER TABLE `_1_providers_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_providers_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_resourceTokens`
--

DROP TABLE IF EXISTS `_1_resourceTokens`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_resourceTokens` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `resourceId` varchar(255) DEFAULT NULL,
  `resourceInternalId` varchar(255) DEFAULT NULL,
  `resourceType` varchar(100) DEFAULT NULL,
  `secret` varchar(512) DEFAULT NULL,
  `expire` datetime(3) DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_expiry_date` (`expire`),
  KEY `_key_accessedAt` (`accessedAt`),
  KEY `_key_resourceInternalId_resourceType` (`resourceInternalId`,`resourceType`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_resourceTokens`
--

LOCK TABLES `_1_resourceTokens` WRITE;
/*!40000 ALTER TABLE `_1_resourceTokens` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_resourceTokens` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_resourceTokens_perms`
--

DROP TABLE IF EXISTS `_1_resourceTokens_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_resourceTokens_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_resourceTokens_perms`
--

LOCK TABLES `_1_resourceTokens_perms` WRITE;
/*!40000 ALTER TABLE `_1_resourceTokens_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_resourceTokens_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_sessions`
--

DROP TABLE IF EXISTS `_1_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_sessions` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `provider` varchar(128) DEFAULT NULL,
  `providerUid` varchar(2048) DEFAULT NULL,
  `providerAccessToken` text DEFAULT NULL,
  `providerAccessTokenExpiry` datetime(3) DEFAULT NULL,
  `providerRefreshToken` text DEFAULT NULL,
  `secret` varchar(512) DEFAULT NULL,
  `userAgent` text DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  `countryCode` varchar(2) DEFAULT NULL,
  `osCode` varchar(256) DEFAULT NULL,
  `osName` varchar(256) DEFAULT NULL,
  `osVersion` varchar(256) DEFAULT NULL,
  `clientType` varchar(256) DEFAULT NULL,
  `clientCode` varchar(256) DEFAULT NULL,
  `clientName` varchar(256) DEFAULT NULL,
  `clientVersion` varchar(256) DEFAULT NULL,
  `clientEngine` varchar(256) DEFAULT NULL,
  `clientEngineVersion` varchar(256) DEFAULT NULL,
  `deviceName` varchar(256) DEFAULT NULL,
  `deviceBrand` varchar(256) DEFAULT NULL,
  `deviceModel` varchar(256) DEFAULT NULL,
  `factors` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`factors`)),
  `expire` datetime(3) DEFAULT NULL,
  `mfaUpdatedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_provider_providerUid` (`provider`,`providerUid`(128)),
  KEY `_key_user` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_sessions`
--

LOCK TABLES `_1_sessions` WRITE;
/*!40000 ALTER TABLE `_1_sessions` DISABLE KEYS */;
INSERT INTO `_1_sessions` VALUES
(9,'69ce3bd314ba8fb2781f','2026-04-02 09:50:11.088','2026-04-02 09:50:11.088','[\"read(\\\"user:69ce3bcd771ae8884a4d\\\")\",\"update(\\\"user:69ce3bcd771ae8884a4d\\\")\",\"delete(\\\"user:69ce3bcd771ae8884a4d\\\")\"]','4','69ce3bcd771ae8884a4d','email','org3@admin.org',NULL,NULL,NULL,'{\"data\":\"09X69B7nOjbWBvOOaP5CYmC09JA6FghnQ3siC2745BCayQMrZidf6k4vfehu6ujpdlMJlsiS0\\/xNgclgk3iAOw==\",\"method\":\"aes-128-gcm\",\"iv\":\"f687a0e06e7e29cc92f6dc80\",\"tag\":\"1a28fdc768a08b57f57d9b17cfdca365\",\"version\":\"1\"}','AppwriteGoSDK/v1.0.0 (linux; amd64)','172.28.0.1','--','LIN','GNU/Linux','','','','','','','','desktop',NULL,NULL,'[\"email\"]','2027-04-02 09:50:11.084',NULL);
/*!40000 ALTER TABLE `_1_sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_sessions_perms`
--

DROP TABLE IF EXISTS `_1_sessions_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_sessions_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=28 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_sessions_perms`
--

LOCK TABLES `_1_sessions_perms` WRITE;
/*!40000 ALTER TABLE `_1_sessions_perms` DISABLE KEYS */;
INSERT INTO `_1_sessions_perms` VALUES
(27,'delete','user:69ce3bcd771ae8884a4d','69ce3bd314ba8fb2781f'),
(25,'read','user:69ce3bcd771ae8884a4d','69ce3bd314ba8fb2781f'),
(26,'update','user:69ce3bcd771ae8884a4d','69ce3bd314ba8fb2781f');
/*!40000 ALTER TABLE `_1_sessions_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_sites`
--

DROP TABLE IF EXISTS `_1_sites`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_sites` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(2048) DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `live` tinyint(1) DEFAULT NULL,
  `installationId` varchar(255) DEFAULT NULL,
  `installationInternalId` varchar(255) DEFAULT NULL,
  `providerRepositoryId` varchar(255) DEFAULT NULL,
  `repositoryId` varchar(255) DEFAULT NULL,
  `repositoryInternalId` varchar(255) DEFAULT NULL,
  `providerBranch` varchar(255) DEFAULT NULL,
  `providerRootDirectory` varchar(255) DEFAULT NULL,
  `providerSilentMode` tinyint(1) DEFAULT NULL,
  `logging` tinyint(1) DEFAULT NULL,
  `framework` varchar(2048) DEFAULT NULL,
  `outputDirectory` text DEFAULT NULL,
  `buildCommand` text DEFAULT NULL,
  `installCommand` text DEFAULT NULL,
  `fallbackFile` text DEFAULT NULL,
  `deploymentInternalId` varchar(255) DEFAULT NULL,
  `deploymentId` varchar(255) DEFAULT NULL,
  `deploymentCreatedAt` datetime(3) DEFAULT NULL,
  `deploymentScreenshotLight` varchar(32) DEFAULT NULL,
  `deploymentScreenshotDark` varchar(32) DEFAULT NULL,
  `latestDeploymentId` varchar(255) DEFAULT NULL,
  `latestDeploymentInternalId` varchar(255) DEFAULT NULL,
  `latestDeploymentCreatedAt` datetime(3) DEFAULT NULL,
  `latestDeploymentStatus` varchar(16) DEFAULT NULL,
  `vars` text DEFAULT NULL,
  `varsProject` text DEFAULT NULL,
  `timeout` int(11) DEFAULT NULL,
  `search` text DEFAULT NULL,
  `specification` varchar(128) DEFAULT NULL,
  `buildRuntime` varchar(2048) DEFAULT NULL,
  `adapter` varchar(16) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_name` (`name`(256)),
  KEY `_key_enabled` (`enabled`),
  KEY `_key_installationId` (`installationId`),
  KEY `_key_installationInternalId` (`installationInternalId`),
  KEY `_key_providerRepositoryId` (`providerRepositoryId`),
  KEY `_key_repositoryId` (`repositoryId`),
  KEY `_key_repositoryInternalId` (`repositoryInternalId`),
  KEY `_key_framework` (`framework`(64)),
  KEY `_key_deploymentId` (`deploymentId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_sites`
--

LOCK TABLES `_1_sites` WRITE;
/*!40000 ALTER TABLE `_1_sites` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_sites` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_sites_perms`
--

DROP TABLE IF EXISTS `_1_sites_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_sites_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_sites_perms`
--

LOCK TABLES `_1_sites_perms` WRITE;
/*!40000 ALTER TABLE `_1_sites_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_sites_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_stats`
--

DROP TABLE IF EXISTS `_1_stats`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_stats` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `metric` varchar(255) DEFAULT NULL,
  `region` varchar(255) DEFAULT NULL,
  `value` bigint(20) DEFAULT NULL,
  `time` datetime(3) DEFAULT NULL,
  `period` varchar(4) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_metric_period_time` (`metric` DESC,`period`,`time`),
  KEY `_key_time` (`time` DESC),
  KEY `_key_period_time` (`period`,`time`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=130 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_stats`
--

LOCK TABLES `_1_stats` WRITE;
/*!40000 ALTER TABLE `_1_stats` DISABLE KEYS */;
INSERT INTO `_1_stats` VALUES
(1,'6465432407243506c29fb6016a7c99d6','2026-04-02 09:39:05.108','2026-04-02 09:39:05.108','[]','buckets','default',1,'2026-04-02 00:00:00.000','1d'),
(2,'e12ab54895b3ff794f8183f2bc46504f','2026-04-02 09:39:05.108','2026-04-02 09:39:05.108','[]','buckets','default',1,'2026-04-02 09:00:00.000','1h'),
(3,'207f9e5f969d4e0fe18414edaff36936','2026-04-02 09:39:05.108','2026-04-02 09:39:05.108','[]','buckets','default',1,NULL,'inf'),
(4,'93ae120ec6e80164401c15f7a0cd58ca','2026-04-02 09:39:42.229','2026-04-02 09:50:16.369','[]','teams','default',3,'2026-04-02 00:00:00.000','1d'),
(5,'dca91900eb35a3466f96cd21530058e2','2026-04-02 09:39:42.229','2026-04-02 09:50:16.369','[]','teams','default',3,'2026-04-02 09:00:00.000','1h'),
(6,'5187567d5608d4c8005d4951749efc1f','2026-04-02 09:39:42.229','2026-04-02 09:50:16.369','[]','teams','default',3,NULL,'inf'),
(10,'0866ce86ddc27710eec07869d841dbaf','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.requests','default',472,'2026-04-02 00:00:00.000','1d'),
(11,'19942d70e671de436b758bbc67c1c599','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.requests','default',472,'2026-04-02 09:00:00.000','1h'),
(12,'cb2e77ca591ce00391bdc519b19d1c9d','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.requests','default',472,NULL,'inf'),
(13,'0e92c1616ae9e6c0d84ca5fe71588d63','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.outbound','default',657633,'2026-04-02 00:00:00.000','1d'),
(14,'3ca2c7da4865ad634c75f01ea42af43c','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.outbound','default',657633,'2026-04-02 09:00:00.000','1h'),
(15,'bb484b186be18594b1610e11ef4d5929','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.outbound','default',657633,NULL,'inf'),
(16,'661b1600b7a13b323f52cfb2bea70f10','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.inbound','default',201259,'2026-04-02 00:00:00.000','1d'),
(17,'aa134569c5d98b7b246865d4f4720840','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.inbound','default',201259,'2026-04-02 09:00:00.000','1h'),
(18,'dd464c3f2219264b190e1ea87ceb2595','2026-04-02 09:42:42.067','2026-04-02 09:52:01.243','[]','network.inbound','default',201259,NULL,'inf'),
(19,'5f92bdffce03058dd0089d271be98148','2026-04-02 09:46:55.540','2026-04-02 09:50:16.369','[]','users','default',4,'2026-04-02 00:00:00.000','1d'),
(20,'69a20292987f79e380f2c0b6d4cb0865','2026-04-02 09:46:55.540','2026-04-02 09:50:16.369','[]','users','default',4,'2026-04-02 09:00:00.000','1h'),
(21,'bfa95da1ed6387963075bceead2e955d','2026-04-02 09:46:55.540','2026-04-02 09:50:16.369','[]','users','default',4,NULL,'inf'),
(43,'88809ea3eef9128be2fb73a460cd7381','2026-04-02 09:48:05.863','2026-04-02 09:50:16.369','[]','sessions','default',1,'2026-04-02 00:00:00.000','1d'),
(44,'69f7e2c31c6aa775d3bfa7c2a38e794c','2026-04-02 09:48:05.863','2026-04-02 09:50:16.369','[]','sessions','default',1,'2026-04-02 09:00:00.000','1h'),
(45,'5b7ef467197280056d7121d8254066ee','2026-04-02 09:48:05.863','2026-04-02 09:50:16.369','[]','sessions','default',1,NULL,'inf');
/*!40000 ALTER TABLE `_1_stats` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_stats_perms`
--

DROP TABLE IF EXISTS `_1_stats_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_stats_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_stats_perms`
--

LOCK TABLES `_1_stats_perms` WRITE;
/*!40000 ALTER TABLE `_1_stats_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_stats_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_subscribers`
--

DROP TABLE IF EXISTS `_1_subscribers`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_subscribers` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `targetId` varchar(255) DEFAULT NULL,
  `targetInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `topicId` varchar(255) DEFAULT NULL,
  `topicInternalId` varchar(255) DEFAULT NULL,
  `providerType` varchar(128) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_unique_target_topic` (`targetInternalId`,`topicInternalId`),
  KEY `_key_targetId` (`targetId`),
  KEY `_key_targetInternalId` (`targetInternalId`),
  KEY `_key_userId` (`userId`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_key_topicId` (`topicId`),
  KEY `_key_topicInternalId` (`topicInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_fulltext_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_subscribers`
--

LOCK TABLES `_1_subscribers` WRITE;
/*!40000 ALTER TABLE `_1_subscribers` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_subscribers` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_subscribers_perms`
--

DROP TABLE IF EXISTS `_1_subscribers_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_subscribers_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_subscribers_perms`
--

LOCK TABLES `_1_subscribers_perms` WRITE;
/*!40000 ALTER TABLE `_1_subscribers_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_subscribers_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_targets`
--

DROP TABLE IF EXISTS `_1_targets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_targets` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `sessionId` varchar(255) DEFAULT NULL,
  `sessionInternalId` varchar(255) DEFAULT NULL,
  `providerType` varchar(255) DEFAULT NULL,
  `providerId` varchar(255) DEFAULT NULL,
  `providerInternalId` varchar(255) DEFAULT NULL,
  `identifier` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `expired` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_identifier` (`identifier`),
  KEY `_key_userId` (`userId`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_key_providerId` (`providerId`),
  KEY `_key_providerInternalId` (`providerInternalId`),
  KEY `_key_expired` (`expired`),
  KEY `_key_session_internal_id` (`sessionInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_targets`
--

LOCK TABLES `_1_targets` WRITE;
/*!40000 ALTER TABLE `_1_targets` DISABLE KEYS */;
INSERT INTO `_1_targets` VALUES
(1,'69ce3a1223aef01bb156','2026-04-02 09:42:42.146','2026-04-02 09:42:42.146','[\"read(\\\"user:64e7705962f0dae3f86d\\\")\",\"update(\\\"user:64e7705962f0dae3f86d\\\")\",\"delete(\\\"user:64e7705962f0dae3f86d\\\")\"]','64e7705962f0dae3f86d','1',NULL,NULL,'email',NULL,NULL,'platform_admin@example.org',NULL,0);
/*!40000 ALTER TABLE `_1_targets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_targets_perms`
--

DROP TABLE IF EXISTS `_1_targets_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_targets_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_targets_perms`
--

LOCK TABLES `_1_targets_perms` WRITE;
/*!40000 ALTER TABLE `_1_targets_perms` DISABLE KEYS */;
INSERT INTO `_1_targets_perms` VALUES
(3,'delete','user:64e7705962f0dae3f86d','69ce3a1223aef01bb156'),
(1,'read','user:64e7705962f0dae3f86d','69ce3a1223aef01bb156'),
(2,'update','user:64e7705962f0dae3f86d','69ce3a1223aef01bb156');
/*!40000 ALTER TABLE `_1_targets_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_teams`
--

DROP TABLE IF EXISTS `_1_teams`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_teams` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `total` int(11) DEFAULT NULL,
  `search` text DEFAULT NULL,
  `prefs` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_name` (`name`),
  KEY `_key_total` (`total`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_teams`
--

LOCK TABLES `_1_teams` WRITE;
/*!40000 ALTER TABLE `_1_teams` DISABLE KEYS */;
INSERT INTO `_1_teams` VALUES
(2,'organization-1','2026-04-02 09:47:13.189','2026-04-02 09:48:25.603','[\"read(\\\"team:organization-1\\\")\",\"update(\\\"team:organization-1\\/owner\\\")\",\"delete(\\\"team:organization-1\\/owner\\\")\"]','Organization 1',2,'organization-1 Organization 1','{\"schemaVersion\":1,\"slug\":\"organization-1\",\"roles\":[{\"slug\":\"chemist\",\"name\":\"Chemist\",\"color\":\"var(--role-blue-bg)\",\"border\":\"var(--role-blue-border)\"},{\"slug\":\"mechanic\",\"name\":\"Mechanic\",\"color\":\"var(--role-blue-bg)\",\"border\":\"var(--role-blue-border)\"},{\"slug\":\"supervisor\",\"name\":\"Supervisor\",\"color\":\"var(--role-cyan-bg)\",\"border\":\"var(--role-cyan-border)\"}]}'),
(3,'organization-2','2026-04-02 09:48:54.613','2026-04-02 09:49:32.913','[\"read(\\\"team:organization-2\\\")\",\"update(\\\"team:organization-2\\/owner\\\")\",\"delete(\\\"team:organization-2\\/owner\\\")\"]','Organization 2',2,'organization-2 Organization 2','{\"schemaVersion\":1,\"slug\":\"organization-2\",\"roles\":[{\"slug\":\"analyst\",\"name\":\"Analyst\",\"color\":\"var(--role-emerald-bg)\",\"border\":\"var(--role-emerald-border)\"}]}'),
(4,'organization-3','2026-04-02 09:49:56.912','2026-04-02 09:50:38.425','[\"read(\\\"team:organization-3\\\")\",\"update(\\\"team:organization-3\\/owner\\\")\",\"delete(\\\"team:organization-3\\/owner\\\")\"]','Organization 3',2,'organization-3 Organization 3','{\"schemaVersion\":1,\"slug\":\"organization-3\",\"roles\":[{\"slug\":\"qainspector\",\"name\":\"QA Inspector\",\"color\":\"var(--role-pink-bg)\",\"border\":\"var(--role-pink-border)\"}]}');
/*!40000 ALTER TABLE `_1_teams` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_teams_perms`
--

DROP TABLE IF EXISTS `_1_teams_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_teams_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=13 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_teams_perms`
--

LOCK TABLES `_1_teams_perms` WRITE;
/*!40000 ALTER TABLE `_1_teams_perms` DISABLE KEYS */;
INSERT INTO `_1_teams_perms` VALUES
(6,'delete','team:organization-1/owner','organization-1'),
(4,'read','team:organization-1','organization-1'),
(5,'update','team:organization-1/owner','organization-1'),
(9,'delete','team:organization-2/owner','organization-2'),
(7,'read','team:organization-2','organization-2'),
(8,'update','team:organization-2/owner','organization-2'),
(12,'delete','team:organization-3/owner','organization-3'),
(10,'read','team:organization-3','organization-3'),
(11,'update','team:organization-3/owner','organization-3');
/*!40000 ALTER TABLE `_1_teams_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_tokens`
--

DROP TABLE IF EXISTS `_1_tokens`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_tokens` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `secret` varchar(512) DEFAULT NULL,
  `expire` datetime(3) DEFAULT NULL,
  `userAgent` text DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_user` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_tokens`
--

LOCK TABLES `_1_tokens` WRITE;
/*!40000 ALTER TABLE `_1_tokens` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_tokens` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_tokens_perms`
--

DROP TABLE IF EXISTS `_1_tokens_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_tokens_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_tokens_perms`
--

LOCK TABLES `_1_tokens_perms` WRITE;
/*!40000 ALTER TABLE `_1_tokens_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_tokens_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_topics`
--

DROP TABLE IF EXISTS `_1_topics`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_topics` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `subscribe` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`subscribe`)),
  `emailTotal` int(11) DEFAULT NULL,
  `smsTotal` int(11) DEFAULT NULL,
  `pushTotal` int(11) DEFAULT NULL,
  `targets` text DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_name` (`name`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_topics`
--

LOCK TABLES `_1_topics` WRITE;
/*!40000 ALTER TABLE `_1_topics` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_topics` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_topics_perms`
--

DROP TABLE IF EXISTS `_1_topics_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_topics_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_topics_perms`
--

LOCK TABLES `_1_topics_perms` WRITE;
/*!40000 ALTER TABLE `_1_topics_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_topics_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_transactionLogs`
--

DROP TABLE IF EXISTS `_1_transactionLogs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_transactionLogs` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `transactionInternalId` varchar(255) DEFAULT NULL,
  `databaseInternalId` varchar(255) DEFAULT NULL,
  `collectionInternalId` varchar(255) DEFAULT NULL,
  `documentId` varchar(255) DEFAULT NULL,
  `action` varchar(32) DEFAULT NULL,
  `data` mediumtext DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_transaction` (`transactionInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_transactionLogs`
--

LOCK TABLES `_1_transactionLogs` WRITE;
/*!40000 ALTER TABLE `_1_transactionLogs` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_transactionLogs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_transactionLogs_perms`
--

DROP TABLE IF EXISTS `_1_transactionLogs_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_transactionLogs_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_transactionLogs_perms`
--

LOCK TABLES `_1_transactionLogs_perms` WRITE;
/*!40000 ALTER TABLE `_1_transactionLogs_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_transactionLogs_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_transactions`
--

DROP TABLE IF EXISTS `_1_transactions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_transactions` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `status` varchar(16) DEFAULT NULL,
  `operations` int(10) unsigned DEFAULT NULL,
  `expiresAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_expiresAt` (`expiresAt` DESC),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_transactions`
--

LOCK TABLES `_1_transactions` WRITE;
/*!40000 ALTER TABLE `_1_transactions` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_transactions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_transactions_perms`
--

DROP TABLE IF EXISTS `_1_transactions_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_transactions_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_transactions_perms`
--

LOCK TABLES `_1_transactions_perms` WRITE;
/*!40000 ALTER TABLE `_1_transactions_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_transactions_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_users`
--

DROP TABLE IF EXISTS `_1_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_users` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `email` varchar(320) DEFAULT NULL,
  `phone` varchar(16) DEFAULT NULL,
  `status` tinyint(1) DEFAULT NULL,
  `labels` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`labels`)),
  `passwordHistory` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`passwordHistory`)),
  `password` text DEFAULT NULL,
  `hash` varchar(256) DEFAULT NULL,
  `hashOptions` text DEFAULT NULL,
  `passwordUpdate` datetime(3) DEFAULT NULL,
  `prefs` text DEFAULT NULL,
  `registration` datetime(3) DEFAULT NULL,
  `emailVerification` tinyint(1) DEFAULT NULL,
  `phoneVerification` tinyint(1) DEFAULT NULL,
  `reset` tinyint(1) DEFAULT NULL,
  `mfa` tinyint(1) DEFAULT NULL,
  `mfaRecoveryCodes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`mfaRecoveryCodes`)),
  `authenticators` text DEFAULT NULL,
  `sessions` text DEFAULT NULL,
  `tokens` text DEFAULT NULL,
  `challenges` text DEFAULT NULL,
  `memberships` text DEFAULT NULL,
  `targets` text DEFAULT NULL,
  `search` text DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  `emailCanonical` varchar(320) DEFAULT NULL,
  `emailIsFree` tinyint(1) DEFAULT NULL,
  `emailIsDisposable` tinyint(1) DEFAULT NULL,
  `emailIsCorporate` tinyint(1) DEFAULT NULL,
  `emailIsCanonical` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_phone` (`phone`),
  UNIQUE KEY `_key_email` (`email`(256)),
  KEY `_key_name` (`name`),
  KEY `_key_status` (`status`),
  KEY `_key_passwordUpdate` (`passwordUpdate`),
  KEY `_key_registration` (`registration`),
  KEY `_key_emailVerification` (`emailVerification`),
  KEY `_key_phoneVerification` (`phoneVerification`),
  KEY `_key_accessedAt` (`accessedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_users`
--

LOCK TABLES `_1_users` WRITE;
/*!40000 ALTER TABLE `_1_users` DISABLE KEYS */;
INSERT INTO `_1_users` VALUES
(1,'64e7705962f0dae3f86d','2026-04-02 09:42:42.132','2026-04-02 09:49:56.953','[\"read(\\\"any\\\")\",\"update(\\\"user:64e7705962f0dae3f86d\\\")\",\"delete(\\\"user:64e7705962f0dae3f86d\\\")\"]','Platform Admin','platform_admin@example.org',NULL,1,'[\"attestaOrgAdmin\"]','[]','{\"data\":\"obbvp0d8VIArcg0Yj6Y4GTZTMBWIqpwDTWZT\\/5PgB3MwipItSEWud377ws7cnUa6iIjRtreRm8Rv4jnVwVcDIgf+1kgY4cOy0TJui4o2M8POE6ICA4Q0dFWBLQ3gY0vHlA==\",\"method\":\"aes-128-gcm\",\"iv\":\"8d51ceccb90de5d40385c546\",\"tag\":\"78a8d7efe20b664a04e7517628f62465\",\"version\":\"1\"}','argon2','{\"type\":\"argon2\",\"memory_cost\":65536,\"time_cost\":4,\"threads\":3}','2026-04-02 09:42:42.130','[]','2026-04-02 09:42:42.130',0,0,0,NULL,'[]',NULL,NULL,NULL,NULL,NULL,NULL,'64e7705962f0dae3f86d platform_admin@example.org Platform Admin label:attestaOrgAdmin','2026-04-02 09:47:13.170','platform_admin@example.org',0,0,1,0),
(2,'69ce3b31064c3cca2338','2026-04-02 09:47:29.026','2026-04-02 09:48:37.936','[\"read(\\\"any\\\")\",\"read(\\\"user:69ce3b31064c3cca2338\\\")\",\"update(\\\"user:69ce3b31064c3cca2338\\\")\",\"delete(\\\"user:69ce3b31064c3cca2338\\\")\"]','org1@admin.org','org1@admin.org',NULL,1,'[\"rchemist\",\"rmechanic\",\"rsupervisor\",\"attestaOrgAdmin\"]','[]','{\"data\":\"iFlFUDPRVZ8EaJtGnbnowisy4dzjj6oCAN2ApneN7CfCSnnETOoLDJvRUbjaFrjNnonZ4ENveoYgFhkIwZaKMrLjfE5oOJEH3Xhkk4jMcuupCsk53H2GO6\\/nJXc00SEi\",\"method\":\"aes-128-gcm\",\"iv\":\"838faba5453bcbed4e3ee8b0\",\"tag\":\"f2dc6c9f758938f10a0a486b5e367edf\",\"version\":\"1\"}','argon2','{\"type\":\"argon2\",\"memory_cost\":7168,\"time_cost\":5,\"threads\":1}','2026-04-02 09:47:42.978','[]','2026-04-02 09:47:29.025',1,NULL,0,NULL,'[]',NULL,NULL,NULL,NULL,NULL,NULL,'69ce3b31064c3cca2338 org1@admin.org org1@admin.org label:rchemist label:rmechanic label:rsupervisor label:attestaOrgAdmin','2026-04-02 09:47:38.012','org1@admin.org',0,0,1,0),
(3,'69ce3b8f57ee7c5a7955','2026-04-02 09:49:03.361','2026-04-02 09:49:37.303','[\"read(\\\"any\\\")\",\"read(\\\"user:69ce3b8f57ee7c5a7955\\\")\",\"update(\\\"user:69ce3b8f57ee7c5a7955\\\")\",\"delete(\\\"user:69ce3b8f57ee7c5a7955\\\")\"]','org2@admin.org','org2@admin.org',NULL,1,'[\"ranalyst\",\"attestaOrgAdmin\"]','[]','{\"data\":\"qt\\/UI2BXxfbJaawwYMrRPooxAkgaBJW9wsj51ySkqewSVebtwPPEQliKnlgYw\\/47rYNiuzMIpBKISvSuPI90tbnThzBO\\/9Q2Zt5u9Oijb3iEeujh7UClKVQhZyx3xWLM\",\"method\":\"aes-128-gcm\",\"iv\":\"fa063aa72987473aec82546b\",\"tag\":\"03031d08ef9dabc378053d1b1c96c9d4\",\"version\":\"1\"}','argon2','{\"type\":\"argon2\",\"memory_cost\":7168,\"time_cost\":5,\"threads\":1}','2026-04-02 09:49:17.699','[]','2026-04-02 09:49:03.360',1,NULL,0,NULL,'[]',NULL,NULL,NULL,NULL,NULL,NULL,'69ce3b8f57ee7c5a7955 org2@admin.org org2@admin.org label:ranalyst label:attestaOrgAdmin','2026-04-02 09:49:13.742','org2@admin.org',0,0,1,0),
(4,'69ce3bcd771ae8884a4d','2026-04-02 09:50:05.489','2026-04-02 09:51:38.047','[\"read(\\\"any\\\")\",\"read(\\\"user:69ce3bcd771ae8884a4d\\\")\",\"update(\\\"user:69ce3bcd771ae8884a4d\\\")\",\"delete(\\\"user:69ce3bcd771ae8884a4d\\\")\"]','org3@admin.org','org3@admin.org',NULL,1,'[\"rqainspector\",\"attestaOrgAdmin\"]','[]','{\"data\":\"RdcdQ+K4Ey02n\\/ZZxi6xumO6Cge\\/riRjUFqHMPP0voMaZjmiYXaumHe\\/0B4a7ah9bMM3lbIEo5BvnfJzQQulkJjwicAFYYUoFGTN6sl5Tncs5Vri+qrcXtkajO3HbMhw\",\"method\":\"aes-128-gcm\",\"iv\":\"c6daf904e091eab3abe23824\",\"tag\":\"4a0f16bae3b0a848f890b89af2cd8bdf\",\"version\":\"1\"}','argon2','{\"type\":\"argon2\",\"memory_cost\":7168,\"time_cost\":5,\"threads\":1}','2026-04-02 09:50:16.397','[]','2026-04-02 09:50:05.487',1,NULL,0,NULL,'[]',NULL,NULL,NULL,NULL,NULL,NULL,'69ce3bcd771ae8884a4d org3@admin.org org3@admin.org label:rqainspector label:attestaOrgAdmin','2026-04-02 09:50:11.124','org3@admin.org',0,0,1,0);
/*!40000 ALTER TABLE `_1_users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_users_perms`
--

DROP TABLE IF EXISTS `_1_users_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_users_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_users_perms`
--

LOCK TABLES `_1_users_perms` WRITE;
/*!40000 ALTER TABLE `_1_users_perms` DISABLE KEYS */;
INSERT INTO `_1_users_perms` VALUES
(3,'delete','user:64e7705962f0dae3f86d','64e7705962f0dae3f86d'),
(1,'read','any','64e7705962f0dae3f86d'),
(2,'update','user:64e7705962f0dae3f86d','64e7705962f0dae3f86d'),
(7,'delete','user:69ce3b31064c3cca2338','69ce3b31064c3cca2338'),
(4,'read','any','69ce3b31064c3cca2338'),
(5,'read','user:69ce3b31064c3cca2338','69ce3b31064c3cca2338'),
(6,'update','user:69ce3b31064c3cca2338','69ce3b31064c3cca2338'),
(11,'delete','user:69ce3b8f57ee7c5a7955','69ce3b8f57ee7c5a7955'),
(8,'read','any','69ce3b8f57ee7c5a7955'),
(9,'read','user:69ce3b8f57ee7c5a7955','69ce3b8f57ee7c5a7955'),
(10,'update','user:69ce3b8f57ee7c5a7955','69ce3b8f57ee7c5a7955'),
(15,'delete','user:69ce3bcd771ae8884a4d','69ce3bcd771ae8884a4d'),
(12,'read','any','69ce3bcd771ae8884a4d'),
(13,'read','user:69ce3bcd771ae8884a4d','69ce3bcd771ae8884a4d'),
(14,'update','user:69ce3bcd771ae8884a4d','69ce3bcd771ae8884a4d');
/*!40000 ALTER TABLE `_1_users_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_variables`
--

DROP TABLE IF EXISTS `_1_variables`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_variables` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `resourceType` varchar(100) DEFAULT NULL,
  `resourceInternalId` varchar(255) DEFAULT NULL,
  `resourceId` varchar(255) DEFAULT NULL,
  `key` varchar(255) DEFAULT NULL,
  `value` varchar(8192) DEFAULT NULL,
  `secret` tinyint(1) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_uniqueKey` (`resourceId`,`key`,`resourceType`),
  KEY `_key_resourceInternalId` (`resourceInternalId`),
  KEY `_key_resourceType` (`resourceType`),
  KEY `_key_resourceId_resourceType` (`resourceId`,`resourceType`),
  KEY `_key_key` (`key`),
  KEY `_key_resource_internal_id_resource_type` (`resourceInternalId`,`resourceType`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_fulltext_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_variables`
--

LOCK TABLES `_1_variables` WRITE;
/*!40000 ALTER TABLE `_1_variables` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_variables` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_1_variables_perms`
--

DROP TABLE IF EXISTS `_1_variables_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_1_variables_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_1_variables_perms`
--

LOCK TABLES `_1_variables_perms` WRITE;
/*!40000 ALTER TABLE `_1_variables_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_1_variables_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console__metadata`
--

DROP TABLE IF EXISTS `_console__metadata`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console__metadata` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `attributes` mediumtext DEFAULT NULL,
  `indexes` mediumtext DEFAULT NULL,
  `documentSecurity` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=33 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console__metadata`
--

LOCK TABLES `_console__metadata` WRITE;
/*!40000 ALTER TABLE `_console__metadata` DISABLE KEYS */;
INSERT INTO `_console__metadata` VALUES
(1,'projects','2026-04-02 09:37:25.121','2026-04-02 09:37:25.121','[\"create(\\\"any\\\")\"]','projects','[{\"$id\":\"teamInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"teamId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"region\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"description\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"database\",\"type\":\"string\",\"size\":256,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"logo\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"url\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"version\",\"type\":\"string\",\"size\":16,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"legalName\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"legalCountry\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"legalState\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"legalCity\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"legalAddress\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"legalTaxId\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"services\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":[],\"format\":\"\"},{\"$id\":\"apis\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":[],\"format\":\"\"},{\"$id\":\"smtp\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\",\"encrypt\"],\"default\":[],\"format\":\"\"},{\"$id\":\"templates\",\"type\":\"string\",\"size\":1000000,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":[],\"format\":\"\"},{\"$id\":\"auths\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":[],\"format\":\"\"},{\"$id\":\"oAuthProviders\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\",\"encrypt\"],\"default\":[],\"format\":\"\"},{\"$id\":\"platforms\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryPlatforms\"],\"default\":null,\"format\":\"\"},{\"$id\":\"webhooks\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryWebhooks\"],\"default\":null,\"format\":\"\"},{\"$id\":\"keys\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryKeys\"],\"default\":null,\"format\":\"\"},{\"$id\":\"devKeys\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryDevKeys\"],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"pingCount\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"},{\"$id\":\"pingedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_team\",\"type\":\"key\",\"attributes\":[\"teamId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_pingCount\",\"type\":\"key\",\"attributes\":[\"pingCount\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_pingedAt\",\"type\":\"key\",\"attributes\":[\"pingedAt\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_database\",\"type\":\"key\",\"attributes\":[\"database\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_region_accessed_at\",\"type\":\"key\",\"attributes\":[\"region\",\"accessedAt\"],\"lengths\":[],\"orders\":[]}]',1),
(2,'schedules','2026-04-02 09:37:25.226','2026-04-02 09:37:25.226','[\"create(\\\"any\\\")\"]','schedules','[{\"$id\":\"resourceType\",\"type\":\"string\",\"size\":100,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"resourceInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"resourceId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"resourceUpdatedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"schedule\",\"type\":\"string\",\"size\":100,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"data\",\"type\":\"string\",\"size\":65535,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\",\"encrypt\"],\"default\":{},\"format\":\"\"},{\"$id\":\"active\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"region\",\"type\":\"string\",\"size\":10,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_region_resourceType_resourceUpdatedAt\",\"type\":\"key\",\"attributes\":[\"region\",\"resourceType\",\"resourceUpdatedAt\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_region_resourceType_projectId_resourceId\",\"type\":\"key\",\"attributes\":[\"region\",\"resourceType\",\"projectId\",\"resourceId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_project_id_region\",\"type\":\"key\",\"attributes\":[\"projectId\",\"region\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_region_rt_active\",\"type\":\"key\",\"attributes\":[\"region\",\"resourceType\",\"active\"],\"lengths\":[],\"orders\":[]}]',1),
(3,'platforms','2026-04-02 09:37:25.311','2026-04-02 09:37:25.311','[\"create(\\\"any\\\")\"]','platforms','[{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"type\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":256,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"key\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"store\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"hostname\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_project\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(4,'keys','2026-04-02 09:37:25.390','2026-04-02 09:37:25.390','[\"create(\\\"any\\\")\"]','keys','[{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"scopes\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"secret\",\"type\":\"string\",\"size\":512,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"expire\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"sdks\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_project\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_accessedAt\",\"type\":\"key\",\"attributes\":[\"accessedAt\"],\"lengths\":[],\"orders\":[]}]',1),
(5,'devKeys','2026-04-02 09:37:25.471','2026-04-02 09:37:25.471','[\"create(\\\"any\\\")\"]','devKeys','[{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"secret\",\"type\":\"string\",\"size\":512,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"expire\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"sdks\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_project\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_accessedAt\",\"type\":\"key\",\"attributes\":[\"accessedAt\"],\"lengths\":[],\"orders\":[]}]',1),
(6,'webhooks','2026-04-02 09:37:25.549','2026-04-02 09:37:25.549','[\"create(\\\"any\\\")\"]','webhooks','[{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"url\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"httpUser\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"httpPass\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"security\",\"type\":\"boolean\",\"size\":0,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"events\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"signatureKey\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"enabled\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":true,\"format\":\"\"},{\"$id\":\"logs\",\"type\":\"string\",\"size\":1000000,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"attempts\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"}]','[{\"$id\":\"_key_project\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(7,'certificates','2026-04-02 09:37:25.638','2026-04-02 09:37:25.638','[\"create(\\\"any\\\")\"]','certificates','[{\"$id\":\"domain\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"issueDate\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"renewDate\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"attempts\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"logs\",\"type\":\"string\",\"size\":1000000,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"updated\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_domain\",\"type\":\"key\",\"attributes\":[\"domain\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(8,'realtime','2026-04-02 09:37:25.723','2026-04-02 09:37:25.723','[\"create(\\\"any\\\")\"]','realtime','[{\"$id\":\"container\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"timestamp\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"value\",\"type\":\"string\",\"size\":16384,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_timestamp\",\"type\":\"key\",\"attributes\":[\"timestamp\"],\"lengths\":[],\"orders\":[\"DESC\"]}]',1),
(9,'rules','2026-04-02 09:37:25.919','2026-04-02 09:37:25.919','[\"create(\\\"any\\\")\"]','rules','[{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"domain\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"type\",\"type\":\"string\",\"size\":32,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"trigger\",\"type\":\"string\",\"size\":32,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"redirectUrl\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"redirectStatusCode\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"deploymentResourceType\",\"type\":\"string\",\"size\":32,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"deploymentId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"deploymentInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"deploymentResourceId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"deploymentResourceInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"deploymentVcsProviderBranch\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"status\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"certificateId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"owner\",\"type\":\"string\",\"size\":16,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"\",\"format\":\"\"},{\"$id\":\"region\",\"type\":\"string\",\"size\":16,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_domain\",\"type\":\"unique\",\"attributes\":[\"domain\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_projectInternalId\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_projectId\",\"type\":\"key\",\"attributes\":[\"projectId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_type\",\"type\":\"key\",\"attributes\":[\"type\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_trigger\",\"type\":\"key\",\"attributes\":[\"trigger\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentResourceType\",\"type\":\"key\",\"attributes\":[\"deploymentResourceType\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentResourceId\",\"type\":\"key\",\"attributes\":[\"deploymentResourceId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentResourceInternalId\",\"type\":\"key\",\"attributes\":[\"deploymentResourceInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentId\",\"type\":\"key\",\"attributes\":[\"deploymentId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentInternalId\",\"type\":\"key\",\"attributes\":[\"deploymentInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_deploymentVcsProviderBranch\",\"type\":\"key\",\"attributes\":[\"deploymentVcsProviderBranch\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_owner\",\"type\":\"key\",\"attributes\":[\"owner\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_region\",\"type\":\"key\",\"attributes\":[\"region\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_piid_riid_rt\",\"type\":\"key\",\"attributes\":[\"projectInternalId\",\"deploymentInternalId\",\"deploymentResourceType\"],\"lengths\":[],\"orders\":[]}]',1),
(10,'installations','2026-04-02 09:37:26.019','2026-04-02 09:37:26.019','[\"create(\\\"any\\\")\"]','installations','[{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerInstallationId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"organization\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"provider\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"personal\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":false,\"format\":\"\"},{\"$id\":\"personalAccessToken\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"personalAccessTokenExpiry\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"personalRefreshToken\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_projectInternalId\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_projectId\",\"type\":\"key\",\"attributes\":[\"projectId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerInstallationId\",\"type\":\"key\",\"attributes\":[\"providerInstallationId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(11,'repositories','2026-04-02 09:37:26.141','2026-04-02 09:37:26.141','[\"create(\\\"any\\\")\"]','repositories','[{\"$id\":\"installationId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"installationInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerRepositoryId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"resourceId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"resourceInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"resourceType\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerPullRequestIds\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_installationId\",\"type\":\"key\",\"attributes\":[\"installationId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_installationInternalId\",\"type\":\"key\",\"attributes\":[\"installationInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_projectInternalId\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_projectId\",\"type\":\"key\",\"attributes\":[\"projectId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerRepositoryId\",\"type\":\"key\",\"attributes\":[\"providerRepositoryId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_resourceId\",\"type\":\"key\",\"attributes\":[\"resourceId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_resourceInternalId\",\"type\":\"key\",\"attributes\":[\"resourceInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_resourceType\",\"type\":\"key\",\"attributes\":[\"resourceType\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_piid_riid_rt\",\"type\":\"key\",\"attributes\":[\"projectInternalId\",\"resourceInternalId\",\"resourceType\"],\"lengths\":[],\"orders\":[]}]',1),
(12,'vcsComments','2026-04-02 09:37:26.262','2026-04-02 09:37:26.262','[\"create(\\\"any\\\")\"]','vcsComments','[{\"$id\":\"installationId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"installationInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"projectInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerRepositoryId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerCommentId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerPullRequestId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerBranch\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_installationId\",\"type\":\"key\",\"attributes\":[\"installationId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_installationInternalId\",\"type\":\"key\",\"attributes\":[\"installationInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_projectInternalId\",\"type\":\"key\",\"attributes\":[\"projectInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_projectId\",\"type\":\"key\",\"attributes\":[\"projectId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerRepositoryId\",\"type\":\"key\",\"attributes\":[\"providerRepositoryId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerPullRequestId\",\"type\":\"key\",\"attributes\":[\"providerPullRequestId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerBranch\",\"type\":\"key\",\"attributes\":[\"providerBranch\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_piid_prid_rt\",\"type\":\"key\",\"attributes\":[\"projectInternalId\",\"providerRepositoryId\"],\"lengths\":[],\"orders\":[]}]',1),
(13,'vcsCommentLocks','2026-04-02 09:37:26.332','2026-04-02 09:37:26.332','[\"create(\\\"any\\\")\"]','vcsCommentLocks','[]','[]',1),
(14,'cache','2026-04-02 09:37:26.411','2026-04-02 09:37:26.411','[\"create(\\\"any\\\")\"]','cache','[{\"$id\":\"resource\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"resourceType\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"mimeType\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"signature\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_accessedAt\",\"type\":\"key\",\"attributes\":[\"accessedAt\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_resource\",\"type\":\"key\",\"attributes\":[\"resource\"],\"lengths\":[],\"orders\":[]}]',1),
(15,'users','2026-04-02 09:37:26.576','2026-04-02 09:37:26.576','[\"create(\\\"any\\\")\"]','users','[{\"$id\":\"name\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"email\",\"type\":\"string\",\"size\":320,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"phone\",\"type\":\"string\",\"size\":16,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"status\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"labels\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"passwordHistory\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"password\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"hash\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"argon2\",\"format\":\"\"},{\"$id\":\"hashOptions\",\"type\":\"string\",\"size\":65535,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":{\"type\":\"argon2\",\"memory_cost\":65536,\"time_cost\":4,\"threads\":3},\"format\":\"\"},{\"$id\":\"passwordUpdate\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"prefs\",\"type\":\"string\",\"size\":65535,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":{},\"format\":\"\"},{\"$id\":\"registration\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"emailVerification\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"phoneVerification\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"reset\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"mfa\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"mfaRecoveryCodes\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[\"encrypt\"],\"default\":[],\"format\":\"\"},{\"$id\":\"authenticators\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryAuthenticators\"],\"default\":null,\"format\":\"\"},{\"$id\":\"sessions\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQuerySessions\"],\"default\":null,\"format\":\"\"},{\"$id\":\"tokens\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryTokens\"],\"default\":null,\"format\":\"\"},{\"$id\":\"challenges\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryChallenges\"],\"default\":null,\"format\":\"\"},{\"$id\":\"memberships\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryMemberships\"],\"default\":null,\"format\":\"\"},{\"$id\":\"targets\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryTargets\"],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"userSearch\"],\"default\":null,\"format\":\"\"},{\"$id\":\"accessedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"emailCanonical\",\"type\":\"string\",\"size\":320,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"emailIsFree\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"emailIsDisposable\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"emailIsCorporate\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"emailIsCanonical\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_email\",\"type\":\"unique\",\"attributes\":[\"email\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_phone\",\"type\":\"unique\",\"attributes\":[\"phone\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_status\",\"type\":\"key\",\"attributes\":[\"status\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_passwordUpdate\",\"type\":\"key\",\"attributes\":[\"passwordUpdate\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_registration\",\"type\":\"key\",\"attributes\":[\"registration\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_emailVerification\",\"type\":\"key\",\"attributes\":[\"emailVerification\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_phoneVerification\",\"type\":\"key\",\"attributes\":[\"phoneVerification\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_accessedAt\",\"type\":\"key\",\"attributes\":[\"accessedAt\"],\"lengths\":[],\"orders\":[]}]',1),
(16,'tokens','2026-04-02 09:37:26.652','2026-04-02 09:37:26.652','[\"create(\\\"any\\\")\"]','tokens','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"type\",\"type\":\"integer\",\"size\":0,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"secret\",\"type\":\"string\",\"size\":512,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"expire\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"userAgent\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"ip\",\"type\":\"string\",\"size\":45,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(17,'authenticators','2026-04-02 09:37:26.726','2026-04-02 09:37:26.726','[\"create(\\\"any\\\")\"]','authenticators','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"type\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"verified\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":false,\"format\":\"\"},{\"$id\":\"data\",\"type\":\"string\",\"size\":65535,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\",\"encrypt\"],\"default\":[],\"format\":\"\"}]','[{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(18,'challenges','2026-04-02 09:37:26.812','2026-04-02 09:37:26.812','[\"create(\\\"any\\\")\"]','challenges','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"type\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"token\",\"type\":\"string\",\"size\":512,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"code\",\"type\":\"string\",\"size\":512,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"expire\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(19,'sessions','2026-04-02 09:37:26.891','2026-04-02 09:37:26.891','[\"create(\\\"any\\\")\"]','sessions','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"provider\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerUid\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerAccessToken\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"providerAccessTokenExpiry\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"providerRefreshToken\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"secret\",\"type\":\"string\",\"size\":512,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"userAgent\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"ip\",\"type\":\"string\",\"size\":45,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"countryCode\",\"type\":\"string\",\"size\":2,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"osCode\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"osName\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"osVersion\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"clientType\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"clientCode\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"clientName\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"clientVersion\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"clientEngine\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"clientEngineVersion\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"deviceName\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"deviceBrand\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"deviceModel\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"factors\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":[],\"format\":\"\"},{\"$id\":\"expire\",\"type\":\"datetime\",\"size\":0,\"required\":true,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"mfaUpdatedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_provider_providerUid\",\"type\":\"key\",\"attributes\":[\"provider\",\"providerUid\"],\"lengths\":[null,128],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]}]',1),
(20,'identities','2026-04-02 09:37:27.004','2026-04-02 09:37:27.004','[\"create(\\\"any\\\")\"]','identities','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"provider\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerUid\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerEmail\",\"type\":\"string\",\"size\":320,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerAccessToken\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"providerAccessTokenExpiry\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"providerRefreshToken\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"secrets\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\",\"encrypt\"],\"default\":[],\"format\":\"\"},{\"$id\":\"scopes\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"expire\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_userInternalId_provider_providerUid\",\"type\":\"unique\",\"attributes\":[\"userInternalId\",\"provider\",\"providerUid\"],\"lengths\":[11,null,128],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_provider_providerUid\",\"type\":\"unique\",\"attributes\":[\"provider\",\"providerUid\"],\"lengths\":[null,128],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_provider\",\"type\":\"key\",\"attributes\":[\"provider\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerUid\",\"type\":\"key\",\"attributes\":[\"providerUid\"],\"lengths\":[255],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerEmail\",\"type\":\"key\",\"attributes\":[\"providerEmail\"],\"lengths\":[255],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerAccessTokenExpiry\",\"type\":\"key\",\"attributes\":[\"providerAccessTokenExpiry\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(21,'teams','2026-04-02 09:37:27.130','2026-04-02 09:37:27.130','[\"create(\\\"any\\\")\"]','teams','[{\"$id\":\"name\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"total\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"prefs\",\"type\":\"string\",\"size\":65535,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":{},\"format\":\"\"}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_total\",\"type\":\"key\",\"attributes\":[\"total\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(22,'memberships','2026-04-02 09:37:27.277','2026-04-02 09:37:27.277','[\"create(\\\"any\\\")\"]','memberships','[{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"teamInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"teamId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"roles\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"invited\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"joined\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"confirm\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"secret\",\"type\":\"string\",\"size\":256,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_unique\",\"type\":\"unique\",\"attributes\":[\"teamInternalId\",\"userInternalId\"],\"lengths\":[null,null],\"orders\":[\"ASC\",\"ASC\"]},{\"$id\":\"_key_user\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_team\",\"type\":\"key\",\"attributes\":[\"teamInternalId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_teamId\",\"type\":\"key\",\"attributes\":[\"teamId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_invited\",\"type\":\"key\",\"attributes\":[\"invited\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_joined\",\"type\":\"key\",\"attributes\":[\"joined\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_confirm\",\"type\":\"key\",\"attributes\":[\"confirm\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(23,'buckets','2026-04-02 09:37:27.437','2026-04-02 09:37:27.437','[\"create(\\\"any\\\")\"]','buckets','[{\"$id\":\"enabled\",\"type\":\"boolean\",\"size\":0,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":128,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"fileSecurity\",\"type\":\"boolean\",\"size\":1,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"maximumFileSize\",\"type\":\"integer\",\"size\":8,\"required\":true,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"allowedFileExtensions\",\"type\":\"string\",\"size\":64,\"required\":true,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"compression\",\"type\":\"string\",\"size\":10,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"encryption\",\"type\":\"boolean\",\"size\":0,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"antivirus\",\"type\":\"boolean\",\"size\":0,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"transformations\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":true,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_fulltext_name\",\"type\":\"fulltext\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_enabled\",\"type\":\"key\",\"attributes\":[\"enabled\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_fileSecurity\",\"type\":\"key\",\"attributes\":[\"fileSecurity\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_maximumFileSize\",\"type\":\"key\",\"attributes\":[\"maximumFileSize\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_encryption\",\"type\":\"key\",\"attributes\":[\"encryption\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_antivirus\",\"type\":\"key\",\"attributes\":[\"antivirus\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(24,'stats','2026-04-02 09:37:27.523','2026-04-02 09:37:27.523','[\"create(\\\"any\\\")\"]','stats','[{\"$id\":\"metric\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"region\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"value\",\"type\":\"integer\",\"size\":8,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"time\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"period\",\"type\":\"string\",\"size\":4,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_time\",\"type\":\"key\",\"attributes\":[\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]},{\"$id\":\"_key_period_time\",\"type\":\"key\",\"attributes\":[\"period\",\"time\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_metric_period_time\",\"type\":\"unique\",\"attributes\":[\"metric\",\"period\",\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]}]',1),
(25,'providers','2026-04-02 09:37:27.667','2026-04-02 09:37:27.667','[\"create(\\\"any\\\")\"]','providers','[{\"$id\":\"name\",\"type\":\"string\",\"size\":128,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"provider\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"type\",\"type\":\"string\",\"size\":128,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"enabled\",\"type\":\"boolean\",\"size\":0,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":true,\"format\":\"\"},{\"$id\":\"credentials\",\"type\":\"string\",\"size\":16384,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[\"json\",\"encrypt\"],\"default\":null,\"format\":\"\"},{\"$id\":\"options\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":[],\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":65535,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"providerSearch\"],\"default\":\"\",\"format\":\"\"}]','[{\"$id\":\"_key_provider\",\"type\":\"key\",\"attributes\":[\"provider\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_name\",\"type\":\"fulltext\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_type\",\"type\":\"key\",\"attributes\":[\"type\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_enabled_type\",\"type\":\"key\",\"attributes\":[\"enabled\",\"type\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]}]',1),
(26,'messages','2026-04-02 09:37:27.785','2026-04-02 09:37:27.785','[\"create(\\\"any\\\")\"]','messages','[{\"$id\":\"providerType\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"status\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":\"processing\",\"format\":\"\"},{\"$id\":\"data\",\"type\":\"string\",\"size\":65535,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":null,\"format\":\"\"},{\"$id\":\"topics\",\"type\":\"string\",\"size\":21845,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":[],\"format\":\"\"},{\"$id\":\"users\",\"type\":\"string\",\"size\":21845,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":[],\"format\":\"\"},{\"$id\":\"targets\",\"type\":\"string\",\"size\":21845,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":[],\"format\":\"\"},{\"$id\":\"scheduledAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"scheduleInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"scheduleId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"deliveredAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"deliveryErrors\",\"type\":\"string\",\"size\":65535,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"deliveredTotal\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"messageSearch\"],\"default\":\"\",\"format\":\"\"}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]}]',1),
(27,'topics','2026-04-02 09:37:27.910','2026-04-02 09:37:27.910','[\"create(\\\"any\\\")\"]','topics','[{\"$id\":\"name\",\"type\":\"string\",\"size\":128,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"subscribe\",\"type\":\"string\",\"size\":128,\"required\":false,\"signed\":true,\"array\":true,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"emailTotal\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"},{\"$id\":\"smsTotal\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"},{\"$id\":\"pushTotal\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":0,\"format\":\"\"},{\"$id\":\"targets\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"subQueryTopicTargets\"],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"topicSearch\"],\"default\":\"\",\"format\":\"\"}]','[{\"$id\":\"_key_name\",\"type\":\"fulltext\",\"attributes\":[\"name\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[\"ASC\"]}]',1),
(28,'subscribers','2026-04-02 09:37:28.063','2026-04-02 09:37:28.063','[\"create(\\\"any\\\")\"]','subscribers','[{\"$id\":\"targetId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"targetInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"topicId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"topicInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerType\",\"type\":\"string\",\"size\":128,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_targetId\",\"type\":\"key\",\"attributes\":[\"targetId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_targetInternalId\",\"type\":\"key\",\"attributes\":[\"targetInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_topicId\",\"type\":\"key\",\"attributes\":[\"topicId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_topicInternalId\",\"type\":\"key\",\"attributes\":[\"topicInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_unique_target_topic\",\"type\":\"unique\",\"attributes\":[\"targetInternalId\",\"topicInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_fulltext_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]}]',1),
(29,'targets','2026-04-02 09:37:28.173','2026-04-02 09:37:28.173','[\"create(\\\"any\\\")\"]','targets','[{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"userInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"sessionId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"sessionInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerType\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"providerInternalId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"identifier\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"expired\",\"type\":\"boolean\",\"size\":0,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":false,\"format\":\"\"}]','[{\"$id\":\"_key_userId\",\"type\":\"key\",\"attributes\":[\"userId\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_userInternalId\",\"type\":\"key\",\"attributes\":[\"userInternalId\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_providerId\",\"type\":\"key\",\"attributes\":[\"providerId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_providerInternalId\",\"type\":\"key\",\"attributes\":[\"providerInternalId\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_identifier\",\"type\":\"unique\",\"attributes\":[\"identifier\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_expired\",\"type\":\"key\",\"attributes\":[\"expired\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_session_internal_id\",\"type\":\"key\",\"attributes\":[\"sessionInternalId\"],\"lengths\":[],\"orders\":[]}]',1),
(30,'audit','2026-04-02 09:37:28.273','2026-04-02 09:37:28.273','[\"create(\\\"any\\\")\"]','audit','[{\"$id\":\"userId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"event\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"resource\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"userAgent\",\"type\":\"string\",\"size\":65534,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"ip\",\"type\":\"string\",\"size\":45,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"location\",\"type\":\"string\",\"size\":45,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[]},{\"$id\":\"time\",\"type\":\"datetime\",\"format\":\"\",\"size\":0,\"signed\":true,\"required\":false,\"array\":false,\"filters\":[\"datetime\"]},{\"$id\":\"data\",\"type\":\"string\",\"size\":16777216,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"]}]','[{\"$id\":\"index2\",\"type\":\"key\",\"attributes\":[\"event\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"index4\",\"type\":\"key\",\"attributes\":[\"userId\",\"event\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"index5\",\"type\":\"key\",\"attributes\":[\"resource\",\"event\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"index-time\",\"type\":\"key\",\"attributes\":[\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]}]',1),
(31,'bucket_1','2026-04-02 09:37:28.480','2026-04-02 09:37:28.480','[\"create(\\\"any\\\")\"]','bucket_1','[{\"$id\":\"bucketId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"bucketInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"path\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"signature\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"mimeType\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"metadata\",\"type\":\"string\",\"size\":75000,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":null,\"format\":\"\"},{\"$id\":\"sizeOriginal\",\"type\":\"integer\",\"size\":8,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"sizeActual\",\"type\":\"integer\",\"size\":8,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"algorithm\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"comment\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLVersion\",\"type\":\"string\",\"size\":64,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLCipher\",\"type\":\"string\",\"size\":64,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLTag\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLIV\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"chunksTotal\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"chunksUploaded\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"transformedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_bucket\",\"type\":\"key\",\"attributes\":[\"bucketId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_signature\",\"type\":\"key\",\"attributes\":[\"signature\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_mimeType\",\"type\":\"key\",\"attributes\":[\"mimeType\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_sizeOriginal\",\"type\":\"key\",\"attributes\":[\"sizeOriginal\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_chunksTotal\",\"type\":\"key\",\"attributes\":[\"chunksTotal\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_chunksUploaded\",\"type\":\"key\",\"attributes\":[\"chunksUploaded\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_transformedAt\",\"type\":\"key\",\"attributes\":[\"transformedAt\"],\"lengths\":[],\"orders\":[]}]',1),
(32,'bucket_2','2026-04-02 09:37:28.635','2026-04-02 09:37:28.635','[\"create(\\\"any\\\")\"]','bucket_2','[{\"$id\":\"bucketId\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"bucketInternalId\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"name\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"path\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"signature\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"mimeType\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"metadata\",\"type\":\"string\",\"size\":75000,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[\"json\"],\"default\":null,\"format\":\"\"},{\"$id\":\"sizeOriginal\",\"type\":\"integer\",\"size\":8,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"sizeActual\",\"type\":\"integer\",\"size\":8,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"algorithm\",\"type\":\"string\",\"size\":255,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"comment\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLVersion\",\"type\":\"string\",\"size\":64,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLCipher\",\"type\":\"string\",\"size\":64,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLTag\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"openSSLIV\",\"type\":\"string\",\"size\":2048,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"chunksTotal\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"chunksUploaded\",\"type\":\"integer\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"transformedAt\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"search\",\"type\":\"string\",\"size\":16384,\"required\":false,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_search\",\"type\":\"fulltext\",\"attributes\":[\"search\"],\"lengths\":[],\"orders\":[]},{\"$id\":\"_key_bucket\",\"type\":\"key\",\"attributes\":[\"bucketId\"],\"lengths\":[null],\"orders\":[\"ASC\"]},{\"$id\":\"_key_name\",\"type\":\"key\",\"attributes\":[\"name\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_signature\",\"type\":\"key\",\"attributes\":[\"signature\"],\"lengths\":[256],\"orders\":[\"ASC\"]},{\"$id\":\"_key_mimeType\",\"type\":\"key\",\"attributes\":[\"mimeType\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_sizeOriginal\",\"type\":\"key\",\"attributes\":[\"sizeOriginal\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_chunksTotal\",\"type\":\"key\",\"attributes\":[\"chunksTotal\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_chunksUploaded\",\"type\":\"key\",\"attributes\":[\"chunksUploaded\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_transformedAt\",\"type\":\"key\",\"attributes\":[\"transformedAt\"],\"lengths\":[],\"orders\":[]}]',1);
/*!40000 ALTER TABLE `_console__metadata` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console__metadata_perms`
--

DROP TABLE IF EXISTS `_console__metadata_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console__metadata_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=33 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console__metadata_perms`
--

LOCK TABLES `_console__metadata_perms` WRITE;
/*!40000 ALTER TABLE `_console__metadata_perms` DISABLE KEYS */;
INSERT INTO `_console__metadata_perms` VALUES
(30,'create','any','audit'),
(17,'create','any','authenticators'),
(23,'create','any','buckets'),
(31,'create','any','bucket_1'),
(32,'create','any','bucket_2'),
(14,'create','any','cache'),
(7,'create','any','certificates'),
(18,'create','any','challenges'),
(5,'create','any','devKeys'),
(20,'create','any','identities'),
(10,'create','any','installations'),
(4,'create','any','keys'),
(22,'create','any','memberships'),
(26,'create','any','messages'),
(3,'create','any','platforms'),
(1,'create','any','projects'),
(25,'create','any','providers'),
(8,'create','any','realtime'),
(11,'create','any','repositories'),
(9,'create','any','rules'),
(2,'create','any','schedules'),
(19,'create','any','sessions'),
(24,'create','any','stats'),
(28,'create','any','subscribers'),
(29,'create','any','targets'),
(21,'create','any','teams'),
(16,'create','any','tokens'),
(27,'create','any','topics'),
(15,'create','any','users'),
(13,'create','any','vcsCommentLocks'),
(12,'create','any','vcsComments'),
(6,'create','any','webhooks');
/*!40000 ALTER TABLE `_console__metadata_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_audit`
--

DROP TABLE IF EXISTS `_console_audit`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_audit` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `event` varchar(255) DEFAULT NULL,
  `resource` varchar(255) DEFAULT NULL,
  `userAgent` text DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  `location` varchar(45) DEFAULT NULL,
  `time` datetime(3) DEFAULT NULL,
  `data` longtext DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `index2` (`event`),
  KEY `index4` (`userId`,`event`),
  KEY `index5` (`resource`,`event`),
  KEY `index-time` (`time` DESC),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=17 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_audit`
--

LOCK TABLES `_console_audit` WRITE;
/*!40000 ALTER TABLE `_console_audit` DISABLE KEYS */;
INSERT INTO `_console_audit` VALUES
(1,'69ce38e7bf9c9418422f','2026-04-02 09:37:43.784','2026-04-02 09:37:43.784','[]','1','user.create','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:37:43.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.737+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":[],\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(2,'69ce38e7daebf24ca4e8','2026-04-02 09:37:43.896','2026-04-02 09:37:43.896','[]','1','session.create','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:37:43.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e7cb822b545d18\",\"$createdAt\":\"2026-04-02T09:37:43.889+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.889+00:00\",\"userId\":\"69ce38e70026eaa5d8db\",\"expire\":\"2027-04-02T09:37:43.833+00:00\",\"provider\":\"email\",\"providerUid\":\"admin@example.org\",\"providerAccessToken\":\"\",\"providerAccessTokenExpiry\":\"\",\"providerRefreshToken\":\"\",\"ip\":\"172.28.0.1\",\"osCode\":\"WIN\",\"osName\":\"Windows\",\"osVersion\":\"10\",\"clientType\":\"browser\",\"clientCode\":\"CH\",\"clientName\":\"Chrome\",\"clientVersion\":\"146.0\",\"clientEngine\":\"Blink\",\"clientEngineVersion\":\"146.0.0.0\",\"deviceName\":\"desktop\",\"deviceBrand\":\"\",\"deviceModel\":\"\",\"countryCode\":\"--\",\"countryName\":\"Unknown\",\"current\":true,\"factors\":[\"password\"],\"secret\":\"\",\"mfaUpdatedAt\":\"\"}}'),
(3,'69ce38e7f3e42fa32082','2026-04-02 09:37:43.998','2026-04-02 09:37:43.998','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:37:43.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.983+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":\"69bd3241001510c17ada\"},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(4,'69ce38e804a610a020b6','2026-04-02 09:37:44.019','2026-04-02 09:37:44.019','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:37:44.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:37:44.011+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":null},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(5,'69ce38e81b77bc3ad403','2026-04-02 09:37:44.112','2026-04-02 09:37:44.112','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:37:44.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:37:44.101+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":\"69bd3241001510c17ada\"},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(6,'69ce38e82a66fad55046','2026-04-02 09:37:44.173','2026-04-02 09:37:44.173','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:37:44.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:37:44.165+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":null},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(7,'69ce38ef436cbf00ee53','2026-04-02 09:37:51.276','2026-04-02 09:37:51.276','[]','1','team.create','team/69ce38ef00104ea230bc','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:37:51.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38ef00104ea230bc\",\"$createdAt\":\"2026-04-02T09:37:51.253+00:00\",\"$updatedAt\":\"2026-04-02T09:37:51.253+00:00\",\"name\":\"Personal projects\",\"total\":1,\"prefs\":[]}}'),
(8,'69ce38fe205981b726ee','2026-04-02 09:38:06.132','2026-04-02 09:38:06.132','[]','1','projects.create','project/attesta','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:38:06.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"attesta\",\"$createdAt\":\"2026-04-02T09:38:02.816+00:00\",\"$updatedAt\":\"2026-04-02T09:38:02.816+00:00\",\"name\":\"attesta\",\"description\":\"\",\"teamId\":\"69ce38ef00104ea230bc\",\"logo\":\"\",\"url\":\"\",\"legalName\":\"\",\"legalCountry\":\"\",\"legalState\":\"\",\"legalCity\":\"\",\"legalAddress\":\"\",\"legalTaxId\":\"\",\"authDuration\":31536000,\"authLimit\":0,\"authSessionsLimit\":10,\"authPasswordHistory\":0,\"authPasswordDictionary\":false,\"authPersonalDataCheck\":false,\"authMockNumbers\":[],\"authSessionAlerts\":false,\"authMembershipsUserName\":false,\"authMembershipsUserEmail\":false,\"authMembershipsMfa\":false,\"authInvalidateSessions\":true,\"oAuthProviders\":[{\"key\":\"amazon\",\"name\":\"Amazon\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"apple\",\"name\":\"Apple\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"auth0\",\"name\":\"Auth0\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"authentik\",\"name\":\"Authentik\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"autodesk\",\"name\":\"Autodesk\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"bitbucket\",\"name\":\"BitBucket\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"bitly\",\"name\":\"Bitly\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"box\",\"name\":\"Box\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"dailymotion\",\"name\":\"Dailymotion\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"discord\",\"name\":\"Discord\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"disqus\",\"name\":\"Disqus\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"dropbox\",\"name\":\"Dropbox\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"etsy\",\"name\":\"Etsy\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"facebook\",\"name\":\"Facebook\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"figma\",\"name\":\"Figma\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"github\",\"name\":\"GitHub\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"gitlab\",\"name\":\"GitLab\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"google\",\"name\":\"Google\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"linkedin\",\"name\":\"LinkedIn\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"microsoft\",\"name\":\"Microsoft\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"notion\",\"name\":\"Notion\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"oidc\",\"name\":\"OpenID Connect\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"okta\",\"name\":\"Okta\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"paypal\",\"name\":\"PayPal\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"paypalSandbox\",\"name\":\"PayPal Sandbox\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"podio\",\"name\":\"Podio\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"salesforce\",\"name\":\"Salesforce\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"slack\",\"name\":\"Slack\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"spotify\",\"name\":\"Spotify\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"stripe\",\"name\":\"Stripe\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"tradeshift\",\"name\":\"Tradeshift\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"tradeshiftBox\",\"name\":\"Tradeshift Sandbox\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"twitch\",\"name\":\"Twitch\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"wordpress\",\"name\":\"WordPress\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"yahoo\",\"name\":\"Yahoo\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"yammer\",\"name\":\"Yammer\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"yandex\",\"name\":\"Yandex\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"zoho\",\"name\":\"Zoho\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"zoom\",\"name\":\"Zoom\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false},{\"key\":\"mock\",\"name\":\"Mock\",\"appId\":\"\",\"secret\":\"\",\"enabled\":false}],\"platforms\":[],\"webhooks\":[],\"keys\":[],\"devKeys\":[],\"smtpEnabled\":false,\"smtpSenderName\":\"\",\"smtpSenderEmail\":\"\",\"smtpReplyTo\":\"\",\"smtpHost\":\"\",\"smtpPort\":\"\",\"smtpUsername\":\"\",\"smtpPassword\":\"\",\"smtpSecure\":\"\",\"pingCount\":0,\"pingedAt\":\"\",\"authEmailPassword\":true,\"authUsersAuthMagicURL\":true,\"authEmailOtp\":true,\"authAnonymous\":true,\"authInvites\":true,\"authJWT\":true,\"authPhone\":true,\"serviceStatusForAccount\":true,\"serviceStatusForAvatars\":true,\"serviceStatusForDatabases\":true,\"serviceStatusForTablesdb\":true,\"serviceStatusForLocale\":true,\"serviceStatusForHealth\":true,\"serviceStatusForStorage\":true,\"serviceStatusForTeams\":true,\"serviceStatusForUsers\":true,\"serviceStatusForSites\":true,\"serviceStatusForFunctions\":true,\"serviceStatusForGraphql\":true,\"serviceStatusForMessaging\":true}}'),
(9,'69ce38fe237b52182e48','2026-04-02 09:38:06.145','2026-04-02 09:38:06.145','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:38:06.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:38:06.137+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":null,\"newOnboardingCompleted\":true},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(10,'69ce39014373af67787e','2026-04-02 09:38:09.276','2026-04-02 09:38:09.276','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:38:09.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:38:09.262+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":\"69ce38ef00104ea230bc\",\"newOnboardingCompleted\":true},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(11,'69ce3901626d66487f57','2026-04-02 09:38:09.403','2026-04-02 09:38:09.403','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:38:09.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:38:09.392+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":\"69ce38ef00104ea230bc\",\"newOnboardingCompleted\":true},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(12,'69ce39016bcee4078cde','2026-04-02 09:38:09.441','2026-04-02 09:38:09.441','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:38:09.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:38:09.432+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":\"69ce38ef00104ea230bc\",\"newOnboardingCompleted\":true},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(13,'69ce392b6c9aa4b98b97','2026-04-02 09:38:51.444','2026-04-02 09:38:51.444','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:38:51.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:38:51.432+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":\"69ce38ef00104ea230bc\",\"newOnboardingCompleted\":true},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(14,'69ce393110d5a59d56d9','2026-04-02 09:38:57.068','2026-04-02 09:38:57.068','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:38:57.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:38:57.046+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":null,\"newOnboardingCompleted\":true,\"console\":{\"\\/(console)\\/project-[region]-[project]\\/storage\":{\"view\":\"table\"}}},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(15,'69ce393aa264016b28d4','2026-04-02 09:39:06.665','2026-04-02 09:39:06.665','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:39:06.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:39:06.643+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":null,\"newOnboardingCompleted\":true,\"console\":{\"\\/(console)\\/project-[region]-[project]\\/storage\":{\"view\":\"table\"},\"\\/(console)\\/project-[region]-[project]\\/databases\":{\"view\":\"table\"}}},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}'),
(16,'69ce3c216af893edc2a2','2026-04-02 09:51:29.438','2026-04-02 09:51:29.438','[]','1','user.update','user/69ce38e70026eaa5d8db','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','','2026-04-02 09:51:29.000','{\"userId\":\"69ce38e70026eaa5d8db\",\"userName\":\"dev admin\",\"userEmail\":\"admin@example.org\",\"userType\":\"user\",\"mode\":\"default\",\"data\":{\"$id\":\"69ce38e70026eaa5d8db\",\"$createdAt\":\"2026-04-02T09:37:43.737+00:00\",\"$updatedAt\":\"2026-04-02T09:51:29.422+00:00\",\"name\":\"dev admin\",\"registration\":\"2026-04-02T09:37:43.724+00:00\",\"status\":true,\"labels\":[],\"passwordUpdate\":\"2026-04-02T09:37:43.724+00:00\",\"email\":\"admin@example.org\",\"phone\":\"\",\"emailVerification\":false,\"phoneVerification\":false,\"mfa\":false,\"prefs\":{\"organization\":\"69ce38ef00104ea230bc\",\"newOnboardingCompleted\":true,\"console\":{\"\\/(console)\\/project-[region]-[project]\\/storage\":{\"view\":\"table\"},\"\\/(console)\\/project-[region]-[project]\\/databases\":{\"view\":\"table\"}}},\"targets\":[{\"$id\":\"69ce38e7bb42fd421be0\",\"$createdAt\":\"2026-04-02T09:37:43.767+00:00\",\"$updatedAt\":\"2026-04-02T09:37:43.767+00:00\",\"name\":\"\",\"userId\":\"69ce38e70026eaa5d8db\",\"providerId\":null,\"providerType\":\"email\",\"identifier\":\"admin@example.org\",\"expired\":false}],\"accessedAt\":\"2026-04-02T09:37:43.724+00:00\"}}');
/*!40000 ALTER TABLE `_console_audit` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_audit_perms`
--

DROP TABLE IF EXISTS `_console_audit_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_audit_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_audit_perms`
--

LOCK TABLES `_console_audit_perms` WRITE;
/*!40000 ALTER TABLE `_console_audit_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_audit_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_authenticators`
--

DROP TABLE IF EXISTS `_console_authenticators`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_authenticators` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `verified` tinyint(1) DEFAULT NULL,
  `data` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_authenticators`
--

LOCK TABLES `_console_authenticators` WRITE;
/*!40000 ALTER TABLE `_console_authenticators` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_authenticators` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_authenticators_perms`
--

DROP TABLE IF EXISTS `_console_authenticators_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_authenticators_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_authenticators_perms`
--

LOCK TABLES `_console_authenticators_perms` WRITE;
/*!40000 ALTER TABLE `_console_authenticators_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_authenticators_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_bucket_1`
--

DROP TABLE IF EXISTS `_console_bucket_1`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_bucket_1` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `bucketId` varchar(255) DEFAULT NULL,
  `bucketInternalId` varchar(255) DEFAULT NULL,
  `name` varchar(2048) DEFAULT NULL,
  `path` varchar(2048) DEFAULT NULL,
  `signature` varchar(2048) DEFAULT NULL,
  `mimeType` varchar(255) DEFAULT NULL,
  `metadata` mediumtext DEFAULT NULL,
  `sizeOriginal` bigint(20) unsigned DEFAULT NULL,
  `sizeActual` bigint(20) unsigned DEFAULT NULL,
  `algorithm` varchar(255) DEFAULT NULL,
  `comment` varchar(2048) DEFAULT NULL,
  `openSSLVersion` varchar(64) DEFAULT NULL,
  `openSSLCipher` varchar(64) DEFAULT NULL,
  `openSSLTag` varchar(2048) DEFAULT NULL,
  `openSSLIV` varchar(2048) DEFAULT NULL,
  `chunksTotal` int(10) unsigned DEFAULT NULL,
  `chunksUploaded` int(10) unsigned DEFAULT NULL,
  `transformedAt` datetime(3) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_bucket` (`bucketId`),
  KEY `_key_name` (`name`(256)),
  KEY `_key_signature` (`signature`(256)),
  KEY `_key_mimeType` (`mimeType`),
  KEY `_key_sizeOriginal` (`sizeOriginal`),
  KEY `_key_chunksTotal` (`chunksTotal`),
  KEY `_key_chunksUploaded` (`chunksUploaded`),
  KEY `_key_transformedAt` (`transformedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_bucket_1`
--

LOCK TABLES `_console_bucket_1` WRITE;
/*!40000 ALTER TABLE `_console_bucket_1` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_bucket_1` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_bucket_1_perms`
--

DROP TABLE IF EXISTS `_console_bucket_1_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_bucket_1_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_bucket_1_perms`
--

LOCK TABLES `_console_bucket_1_perms` WRITE;
/*!40000 ALTER TABLE `_console_bucket_1_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_bucket_1_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_bucket_2`
--

DROP TABLE IF EXISTS `_console_bucket_2`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_bucket_2` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `bucketId` varchar(255) DEFAULT NULL,
  `bucketInternalId` varchar(255) DEFAULT NULL,
  `name` varchar(2048) DEFAULT NULL,
  `path` varchar(2048) DEFAULT NULL,
  `signature` varchar(2048) DEFAULT NULL,
  `mimeType` varchar(255) DEFAULT NULL,
  `metadata` mediumtext DEFAULT NULL,
  `sizeOriginal` bigint(20) unsigned DEFAULT NULL,
  `sizeActual` bigint(20) unsigned DEFAULT NULL,
  `algorithm` varchar(255) DEFAULT NULL,
  `comment` varchar(2048) DEFAULT NULL,
  `openSSLVersion` varchar(64) DEFAULT NULL,
  `openSSLCipher` varchar(64) DEFAULT NULL,
  `openSSLTag` varchar(2048) DEFAULT NULL,
  `openSSLIV` varchar(2048) DEFAULT NULL,
  `chunksTotal` int(10) unsigned DEFAULT NULL,
  `chunksUploaded` int(10) unsigned DEFAULT NULL,
  `transformedAt` datetime(3) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_bucket` (`bucketId`),
  KEY `_key_name` (`name`(256)),
  KEY `_key_signature` (`signature`(256)),
  KEY `_key_mimeType` (`mimeType`),
  KEY `_key_sizeOriginal` (`sizeOriginal`),
  KEY `_key_chunksTotal` (`chunksTotal`),
  KEY `_key_chunksUploaded` (`chunksUploaded`),
  KEY `_key_transformedAt` (`transformedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_bucket_2`
--

LOCK TABLES `_console_bucket_2` WRITE;
/*!40000 ALTER TABLE `_console_bucket_2` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_bucket_2` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_bucket_2_perms`
--

DROP TABLE IF EXISTS `_console_bucket_2_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_bucket_2_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_bucket_2_perms`
--

LOCK TABLES `_console_bucket_2_perms` WRITE;
/*!40000 ALTER TABLE `_console_bucket_2_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_bucket_2_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_buckets`
--

DROP TABLE IF EXISTS `_console_buckets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_buckets` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `fileSecurity` tinyint(1) DEFAULT NULL,
  `maximumFileSize` bigint(20) unsigned DEFAULT NULL,
  `allowedFileExtensions` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`allowedFileExtensions`)),
  `compression` varchar(10) DEFAULT NULL,
  `encryption` tinyint(1) DEFAULT NULL,
  `antivirus` tinyint(1) DEFAULT NULL,
  `transformations` tinyint(1) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_enabled` (`enabled`),
  KEY `_key_name` (`name`),
  KEY `_key_fileSecurity` (`fileSecurity`),
  KEY `_key_maximumFileSize` (`maximumFileSize`),
  KEY `_key_encryption` (`encryption`),
  KEY `_key_antivirus` (`antivirus`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_fulltext_name` (`name`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_buckets`
--

LOCK TABLES `_console_buckets` WRITE;
/*!40000 ALTER TABLE `_console_buckets` DISABLE KEYS */;
INSERT INTO `_console_buckets` VALUES
(1,'default','2026-04-02 09:37:28.277','2026-04-02 09:37:28.277','[\"create(\\\"any\\\")\",\"read(\\\"any\\\")\",\"update(\\\"any\\\")\",\"delete(\\\"any\\\")\"]',1,'Default',1,30000000,'[]','gzip',1,1,1,'buckets Default'),
(2,'screenshots','2026-04-02 09:37:28.486','2026-04-02 09:37:28.486','[]',1,'Screenshots',1,20000000,'[\"png\"]','gzip',0,0,1,'buckets Screenshots');
/*!40000 ALTER TABLE `_console_buckets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_buckets_perms`
--

DROP TABLE IF EXISTS `_console_buckets_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_buckets_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_buckets_perms`
--

LOCK TABLES `_console_buckets_perms` WRITE;
/*!40000 ALTER TABLE `_console_buckets_perms` DISABLE KEYS */;
INSERT INTO `_console_buckets_perms` VALUES
(1,'create','any','default'),
(4,'delete','any','default'),
(2,'read','any','default'),
(3,'update','any','default');
/*!40000 ALTER TABLE `_console_buckets_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_cache`
--

DROP TABLE IF EXISTS `_console_cache`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_cache` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `resource` varchar(255) DEFAULT NULL,
  `resourceType` varchar(255) DEFAULT NULL,
  `mimeType` varchar(255) DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  `signature` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_accessedAt` (`accessedAt`),
  KEY `_key_resource` (`resource`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_cache`
--

LOCK TABLES `_console_cache` WRITE;
/*!40000 ALTER TABLE `_console_cache` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_cache` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_cache_perms`
--

DROP TABLE IF EXISTS `_console_cache_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_cache_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_cache_perms`
--

LOCK TABLES `_console_cache_perms` WRITE;
/*!40000 ALTER TABLE `_console_cache_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_cache_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_certificates`
--

DROP TABLE IF EXISTS `_console_certificates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_certificates` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `domain` varchar(255) DEFAULT NULL,
  `issueDate` datetime(3) DEFAULT NULL,
  `renewDate` datetime(3) DEFAULT NULL,
  `attempts` int(11) DEFAULT NULL,
  `logs` mediumtext DEFAULT NULL,
  `updated` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_domain` (`domain`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_certificates`
--

LOCK TABLES `_console_certificates` WRITE;
/*!40000 ALTER TABLE `_console_certificates` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_certificates` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_certificates_perms`
--

DROP TABLE IF EXISTS `_console_certificates_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_certificates_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_certificates_perms`
--

LOCK TABLES `_console_certificates_perms` WRITE;
/*!40000 ALTER TABLE `_console_certificates_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_certificates_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_challenges`
--

DROP TABLE IF EXISTS `_console_challenges`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_challenges` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `token` varchar(512) DEFAULT NULL,
  `code` varchar(512) DEFAULT NULL,
  `expire` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_user` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_challenges`
--

LOCK TABLES `_console_challenges` WRITE;
/*!40000 ALTER TABLE `_console_challenges` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_challenges` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_challenges_perms`
--

DROP TABLE IF EXISTS `_console_challenges_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_challenges_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_challenges_perms`
--

LOCK TABLES `_console_challenges_perms` WRITE;
/*!40000 ALTER TABLE `_console_challenges_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_challenges_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_devKeys`
--

DROP TABLE IF EXISTS `_console_devKeys`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_devKeys` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `secret` varchar(512) DEFAULT NULL,
  `expire` datetime(3) DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  `sdks` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`sdks`)),
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_project` (`projectInternalId`),
  KEY `_key_accessedAt` (`accessedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_devKeys`
--

LOCK TABLES `_console_devKeys` WRITE;
/*!40000 ALTER TABLE `_console_devKeys` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_devKeys` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_devKeys_perms`
--

DROP TABLE IF EXISTS `_console_devKeys_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_devKeys_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_devKeys_perms`
--

LOCK TABLES `_console_devKeys_perms` WRITE;
/*!40000 ALTER TABLE `_console_devKeys_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_devKeys_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_identities`
--

DROP TABLE IF EXISTS `_console_identities`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_identities` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `provider` varchar(128) DEFAULT NULL,
  `providerUid` varchar(2048) DEFAULT NULL,
  `providerEmail` varchar(320) DEFAULT NULL,
  `providerAccessToken` text DEFAULT NULL,
  `providerAccessTokenExpiry` datetime(3) DEFAULT NULL,
  `providerRefreshToken` text DEFAULT NULL,
  `secrets` text DEFAULT NULL,
  `scopes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`scopes`)),
  `expire` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_userInternalId_provider_providerUid` (`userInternalId`(11),`provider`,`providerUid`(128)),
  UNIQUE KEY `_key_provider_providerUid` (`provider`,`providerUid`(128)),
  KEY `_key_userId` (`userId`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_key_provider` (`provider`),
  KEY `_key_providerUid` (`providerUid`(255)),
  KEY `_key_providerEmail` (`providerEmail`(255)),
  KEY `_key_providerAccessTokenExpiry` (`providerAccessTokenExpiry`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_identities`
--

LOCK TABLES `_console_identities` WRITE;
/*!40000 ALTER TABLE `_console_identities` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_identities` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_identities_perms`
--

DROP TABLE IF EXISTS `_console_identities_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_identities_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_identities_perms`
--

LOCK TABLES `_console_identities_perms` WRITE;
/*!40000 ALTER TABLE `_console_identities_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_identities_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_installations`
--

DROP TABLE IF EXISTS `_console_installations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_installations` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `providerInstallationId` varchar(255) DEFAULT NULL,
  `organization` varchar(255) DEFAULT NULL,
  `provider` varchar(255) DEFAULT NULL,
  `personal` tinyint(1) DEFAULT NULL,
  `personalAccessToken` varchar(256) DEFAULT NULL,
  `personalAccessTokenExpiry` datetime(3) DEFAULT NULL,
  `personalRefreshToken` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_projectInternalId` (`projectInternalId`),
  KEY `_key_projectId` (`projectId`),
  KEY `_key_providerInstallationId` (`providerInstallationId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_installations`
--

LOCK TABLES `_console_installations` WRITE;
/*!40000 ALTER TABLE `_console_installations` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_installations` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_installations_perms`
--

DROP TABLE IF EXISTS `_console_installations_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_installations_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_installations_perms`
--

LOCK TABLES `_console_installations_perms` WRITE;
/*!40000 ALTER TABLE `_console_installations_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_installations_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_keys`
--

DROP TABLE IF EXISTS `_console_keys`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_keys` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `scopes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`scopes`)),
  `secret` varchar(512) DEFAULT NULL,
  `expire` datetime(3) DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  `sdks` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`sdks`)),
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_project` (`projectInternalId`),
  KEY `_key_accessedAt` (`accessedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_keys`
--

LOCK TABLES `_console_keys` WRITE;
/*!40000 ALTER TABLE `_console_keys` DISABLE KEYS */;
INSERT INTO `_console_keys` VALUES
(1,'69ce392b5619eb0e9dad','2026-04-02 09:38:51.352','2026-04-02 09:42:42.045','[\"read(\\\"any\\\")\",\"update(\\\"any\\\")\",\"delete(\\\"any\\\")\"]','1','attesta','attesta auth key','[\"sessions.write\",\"users.read\",\"users.write\",\"teams.read\",\"teams.write\",\"files.read\",\"files.write\"]','{\"data\":\"1nxzwWT4Qj878FGZPVTG\\/5RtumRiYQd4xi2DFEZZt9PlrR8kaeIWb2ORfLxuJ1gh3miA9gNld6jb3evdQaWeTDOzjFSIwMY1bekd2XAauSJZcotEvxpnekzxF9Kcwr7pZGLsbopLQ7VPHNrTBrKVjio\\/U+OSSrgXWQCMFkaxNWag8vJc1FOVQY9jZi9SOXi60pXbXCNxp8uvgfyW1RfJ9e+Th4Te8FkyqTJ\\/qFYUZhfe4xym32fe2\\/FQsEs9SSRpKpsgD14HkPd7jr0poig\\/dtaZ92FP08cu3l48Cmtxv59JDbv5vnOs0xnP30+QVyhYpVFxlIT6wsDDq7w\\/6frvoyBZVwLf8AHHkA==\",\"method\":\"aes-128-gcm\",\"iv\":\"fe3a418f8b13e27f06defdd0\",\"tag\":\"ea434598e931376f13e089a35e59239c\",\"version\":\"1\"}',NULL,'2026-04-02 09:42:42.045','[]');
/*!40000 ALTER TABLE `_console_keys` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_keys_perms`
--

DROP TABLE IF EXISTS `_console_keys_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_keys_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_keys_perms`
--

LOCK TABLES `_console_keys_perms` WRITE;
/*!40000 ALTER TABLE `_console_keys_perms` DISABLE KEYS */;
INSERT INTO `_console_keys_perms` VALUES
(3,'delete','any','69ce392b5619eb0e9dad'),
(1,'read','any','69ce392b5619eb0e9dad'),
(2,'update','any','69ce392b5619eb0e9dad');
/*!40000 ALTER TABLE `_console_keys_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_memberships`
--

DROP TABLE IF EXISTS `_console_memberships`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_memberships` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `teamInternalId` varchar(255) DEFAULT NULL,
  `teamId` varchar(255) DEFAULT NULL,
  `roles` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`roles`)),
  `invited` datetime(3) DEFAULT NULL,
  `joined` datetime(3) DEFAULT NULL,
  `confirm` tinyint(1) DEFAULT NULL,
  `secret` varchar(256) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_unique` (`teamInternalId`,`userInternalId`),
  KEY `_key_user` (`userInternalId`),
  KEY `_key_team` (`teamInternalId`),
  KEY `_key_userId` (`userId`),
  KEY `_key_teamId` (`teamId`),
  KEY `_key_invited` (`invited`),
  KEY `_key_joined` (`joined`),
  KEY `_key_confirm` (`confirm`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_memberships`
--

LOCK TABLES `_console_memberships` WRITE;
/*!40000 ALTER TABLE `_console_memberships` DISABLE KEYS */;
INSERT INTO `_console_memberships` VALUES
(1,'69ce38ef41fc4a5bc11e','2026-04-02 09:37:51.270','2026-04-02 09:37:51.270','[\"read(\\\"user:69ce38e70026eaa5d8db\\\")\",\"read(\\\"team:69ce38ef00104ea230bc\\\")\",\"update(\\\"user:69ce38e70026eaa5d8db\\\")\",\"update(\\\"team:69ce38ef00104ea230bc\\/owner\\\")\",\"delete(\\\"user:69ce38e70026eaa5d8db\\\")\",\"delete(\\\"team:69ce38ef00104ea230bc\\/owner\\\")\"]','1','69ce38e70026eaa5d8db','1','69ce38ef00104ea230bc','[\"owner\"]','2026-04-02 09:37:51.270','2026-04-02 09:37:51.270',1,'{\"data\":\"\",\"method\":\"aes-128-gcm\",\"iv\":\"b08e74f27539fa33a9990bc8\",\"tag\":\"0cfd6776f6fdb954096cd2183847c564\",\"version\":\"1\"}','69ce38ef41fc4a5bc11e 69ce38e70026eaa5d8db');
/*!40000 ALTER TABLE `_console_memberships` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_memberships_perms`
--

DROP TABLE IF EXISTS `_console_memberships_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_memberships_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_memberships_perms`
--

LOCK TABLES `_console_memberships_perms` WRITE;
/*!40000 ALTER TABLE `_console_memberships_perms` DISABLE KEYS */;
INSERT INTO `_console_memberships_perms` VALUES
(6,'delete','team:69ce38ef00104ea230bc/owner','69ce38ef41fc4a5bc11e'),
(5,'delete','user:69ce38e70026eaa5d8db','69ce38ef41fc4a5bc11e'),
(2,'read','team:69ce38ef00104ea230bc','69ce38ef41fc4a5bc11e'),
(1,'read','user:69ce38e70026eaa5d8db','69ce38ef41fc4a5bc11e'),
(4,'update','team:69ce38ef00104ea230bc/owner','69ce38ef41fc4a5bc11e'),
(3,'update','user:69ce38e70026eaa5d8db','69ce38ef41fc4a5bc11e');
/*!40000 ALTER TABLE `_console_memberships_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_messages`
--

DROP TABLE IF EXISTS `_console_messages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_messages` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `providerType` varchar(255) DEFAULT NULL,
  `status` varchar(255) DEFAULT NULL,
  `data` text DEFAULT NULL,
  `topics` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`topics`)),
  `users` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`users`)),
  `targets` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`targets`)),
  `scheduledAt` datetime(3) DEFAULT NULL,
  `scheduleInternalId` varchar(255) DEFAULT NULL,
  `scheduleId` varchar(255) DEFAULT NULL,
  `deliveredAt` datetime(3) DEFAULT NULL,
  `deliveryErrors` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`deliveryErrors`)),
  `deliveredTotal` int(11) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_messages`
--

LOCK TABLES `_console_messages` WRITE;
/*!40000 ALTER TABLE `_console_messages` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_messages` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_messages_perms`
--

DROP TABLE IF EXISTS `_console_messages_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_messages_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_messages_perms`
--

LOCK TABLES `_console_messages_perms` WRITE;
/*!40000 ALTER TABLE `_console_messages_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_messages_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_platforms`
--

DROP TABLE IF EXISTS `_console_platforms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_platforms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `type` varchar(255) DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `key` varchar(255) DEFAULT NULL,
  `store` varchar(256) DEFAULT NULL,
  `hostname` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_project` (`projectInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_platforms`
--

LOCK TABLES `_console_platforms` WRITE;
/*!40000 ALTER TABLE `_console_platforms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_platforms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_platforms_perms`
--

DROP TABLE IF EXISTS `_console_platforms_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_platforms_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_platforms_perms`
--

LOCK TABLES `_console_platforms_perms` WRITE;
/*!40000 ALTER TABLE `_console_platforms_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_platforms_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_projects`
--

DROP TABLE IF EXISTS `_console_projects`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_projects` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `teamInternalId` varchar(255) DEFAULT NULL,
  `teamId` varchar(255) DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `region` varchar(128) DEFAULT NULL,
  `description` varchar(256) DEFAULT NULL,
  `database` varchar(256) DEFAULT NULL,
  `logo` varchar(255) DEFAULT NULL,
  `url` text DEFAULT NULL,
  `version` varchar(16) DEFAULT NULL,
  `legalName` varchar(256) DEFAULT NULL,
  `legalCountry` varchar(256) DEFAULT NULL,
  `legalState` varchar(256) DEFAULT NULL,
  `legalCity` varchar(256) DEFAULT NULL,
  `legalAddress` varchar(256) DEFAULT NULL,
  `legalTaxId` varchar(256) DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  `services` text DEFAULT NULL,
  `apis` text DEFAULT NULL,
  `smtp` text DEFAULT NULL,
  `templates` mediumtext DEFAULT NULL,
  `auths` text DEFAULT NULL,
  `oAuthProviders` text DEFAULT NULL,
  `platforms` text DEFAULT NULL,
  `webhooks` text DEFAULT NULL,
  `keys` text DEFAULT NULL,
  `devKeys` text DEFAULT NULL,
  `search` text DEFAULT NULL,
  `pingCount` int(10) unsigned DEFAULT NULL,
  `pingedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_name` (`name`),
  KEY `_key_team` (`teamId`),
  KEY `_key_pingCount` (`pingCount`),
  KEY `_key_pingedAt` (`pingedAt`),
  KEY `_key_database` (`database`),
  KEY `_key_region_accessed_at` (`region`,`accessedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_projects`
--

LOCK TABLES `_console_projects` WRITE;
/*!40000 ALTER TABLE `_console_projects` DISABLE KEYS */;
INSERT INTO `_console_projects` VALUES
(1,'attesta','2026-04-02 09:38:02.816','2026-04-02 09:38:02.816','[\"read(\\\"team:69ce38ef00104ea230bc\\\")\",\"update(\\\"team:69ce38ef00104ea230bc\\/owner\\\")\",\"update(\\\"team:69ce38ef00104ea230bc\\/developer\\\")\",\"delete(\\\"team:69ce38ef00104ea230bc\\/owner\\\")\",\"delete(\\\"team:69ce38ef00104ea230bc\\/developer\\\")\"]','1','69ce38ef00104ea230bc','attesta','default','','database_db_main','','','1.8.1','','','','','','','2026-04-02 09:38:02.816','{}','[]','{\"data\":\"ues=\",\"method\":\"aes-128-gcm\",\"iv\":\"7f522772d7a5ba0f9c99324b\",\"tag\":\"525989fbd82b6084a141a27fe591e3be\",\"version\":\"1\"}','[]','{\"limit\":0,\"maxSessions\":10,\"passwordHistory\":0,\"passwordDictionary\":false,\"duration\":31536000,\"personalDataCheck\":false,\"mockNumbers\":[],\"sessionAlerts\":false,\"membershipsUserName\":false,\"membershipsUserEmail\":false,\"membershipsMfa\":false,\"invalidateSessions\":true,\"emailPassword\":true,\"usersAuthMagicURL\":true,\"emailOtp\":true,\"anonymous\":true,\"invites\":true,\"JWT\":true,\"phone\":true}','{\"data\":\"b0s=\",\"method\":\"aes-128-gcm\",\"iv\":\"27326fb1ca025777840f0fb6\",\"tag\":\"4fdcfe5fc55a55e1c7836ba778635de6\",\"version\":\"1\"}',NULL,NULL,NULL,NULL,'attesta attesta',0,NULL);
/*!40000 ALTER TABLE `_console_projects` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_projects_perms`
--

DROP TABLE IF EXISTS `_console_projects_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_projects_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_projects_perms`
--

LOCK TABLES `_console_projects_perms` WRITE;
/*!40000 ALTER TABLE `_console_projects_perms` DISABLE KEYS */;
INSERT INTO `_console_projects_perms` VALUES
(5,'delete','team:69ce38ef00104ea230bc/developer','attesta'),
(4,'delete','team:69ce38ef00104ea230bc/owner','attesta'),
(1,'read','team:69ce38ef00104ea230bc','attesta'),
(3,'update','team:69ce38ef00104ea230bc/developer','attesta'),
(2,'update','team:69ce38ef00104ea230bc/owner','attesta');
/*!40000 ALTER TABLE `_console_projects_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_providers`
--

DROP TABLE IF EXISTS `_console_providers`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_providers` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `provider` varchar(255) DEFAULT NULL,
  `type` varchar(128) DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `credentials` text DEFAULT NULL,
  `options` text DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_provider` (`provider`),
  KEY `_key_type` (`type`),
  KEY `_key_enabled_type` (`enabled`,`type`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_name` (`name`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_providers`
--

LOCK TABLES `_console_providers` WRITE;
/*!40000 ALTER TABLE `_console_providers` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_providers` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_providers_perms`
--

DROP TABLE IF EXISTS `_console_providers_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_providers_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_providers_perms`
--

LOCK TABLES `_console_providers_perms` WRITE;
/*!40000 ALTER TABLE `_console_providers_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_providers_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_realtime`
--

DROP TABLE IF EXISTS `_console_realtime`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_realtime` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `container` varchar(255) DEFAULT NULL,
  `timestamp` datetime(3) DEFAULT NULL,
  `value` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_timestamp` (`timestamp` DESC),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_realtime`
--

LOCK TABLES `_console_realtime` WRITE;
/*!40000 ALTER TABLE `_console_realtime` DISABLE KEYS */;
INSERT INTO `_console_realtime` VALUES
(1,'69ce38d679213268ea19','2026-04-02 09:37:26.497','2026-04-02 09:57:43.493','[]','69ce38ce75ed8','2026-04-02 09:57:43.492','{\"console\":3}');
/*!40000 ALTER TABLE `_console_realtime` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_realtime_perms`
--

DROP TABLE IF EXISTS `_console_realtime_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_realtime_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_realtime_perms`
--

LOCK TABLES `_console_realtime_perms` WRITE;
/*!40000 ALTER TABLE `_console_realtime_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_realtime_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_repositories`
--

DROP TABLE IF EXISTS `_console_repositories`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_repositories` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `installationId` varchar(255) DEFAULT NULL,
  `installationInternalId` varchar(255) DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `providerRepositoryId` varchar(255) DEFAULT NULL,
  `resourceId` varchar(255) DEFAULT NULL,
  `resourceInternalId` varchar(255) DEFAULT NULL,
  `resourceType` varchar(255) DEFAULT NULL,
  `providerPullRequestIds` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`providerPullRequestIds`)),
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_installationId` (`installationId`),
  KEY `_key_installationInternalId` (`installationInternalId`),
  KEY `_key_projectInternalId` (`projectInternalId`),
  KEY `_key_projectId` (`projectId`),
  KEY `_key_providerRepositoryId` (`providerRepositoryId`),
  KEY `_key_resourceId` (`resourceId`),
  KEY `_key_resourceInternalId` (`resourceInternalId`),
  KEY `_key_resourceType` (`resourceType`),
  KEY `_key_piid_riid_rt` (`projectInternalId`,`resourceInternalId`,`resourceType`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_repositories`
--

LOCK TABLES `_console_repositories` WRITE;
/*!40000 ALTER TABLE `_console_repositories` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_repositories` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_repositories_perms`
--

DROP TABLE IF EXISTS `_console_repositories_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_repositories_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_repositories_perms`
--

LOCK TABLES `_console_repositories_perms` WRITE;
/*!40000 ALTER TABLE `_console_repositories_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_repositories_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_rules`
--

DROP TABLE IF EXISTS `_console_rules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_rules` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `domain` varchar(255) DEFAULT NULL,
  `type` varchar(32) DEFAULT NULL,
  `trigger` varchar(32) DEFAULT NULL,
  `redirectUrl` varchar(2048) DEFAULT NULL,
  `redirectStatusCode` int(11) DEFAULT NULL,
  `deploymentResourceType` varchar(32) DEFAULT NULL,
  `deploymentId` varchar(255) DEFAULT NULL,
  `deploymentInternalId` varchar(255) DEFAULT NULL,
  `deploymentResourceId` varchar(255) DEFAULT NULL,
  `deploymentResourceInternalId` varchar(255) DEFAULT NULL,
  `deploymentVcsProviderBranch` varchar(255) DEFAULT NULL,
  `status` varchar(255) DEFAULT NULL,
  `certificateId` varchar(255) DEFAULT NULL,
  `search` text DEFAULT NULL,
  `owner` varchar(16) DEFAULT NULL,
  `region` varchar(16) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_domain` (`domain`),
  KEY `_key_projectInternalId` (`projectInternalId`),
  KEY `_key_projectId` (`projectId`),
  KEY `_key_type` (`type`),
  KEY `_key_trigger` (`trigger`),
  KEY `_key_deploymentResourceType` (`deploymentResourceType`),
  KEY `_key_deploymentResourceId` (`deploymentResourceId`),
  KEY `_key_deploymentResourceInternalId` (`deploymentResourceInternalId`),
  KEY `_key_deploymentId` (`deploymentId`),
  KEY `_key_deploymentInternalId` (`deploymentInternalId`),
  KEY `_key_deploymentVcsProviderBranch` (`deploymentVcsProviderBranch`),
  KEY `_key_owner` (`owner`),
  KEY `_key_region` (`region`),
  KEY `_key_piid_riid_rt` (`projectInternalId`,`deploymentInternalId`,`deploymentResourceType`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_rules`
--

LOCK TABLES `_console_rules` WRITE;
/*!40000 ALTER TABLE `_console_rules` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_rules` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_rules_perms`
--

DROP TABLE IF EXISTS `_console_rules_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_rules_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_rules_perms`
--

LOCK TABLES `_console_rules_perms` WRITE;
/*!40000 ALTER TABLE `_console_rules_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_rules_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_schedules`
--

DROP TABLE IF EXISTS `_console_schedules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_schedules` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `resourceType` varchar(100) DEFAULT NULL,
  `resourceInternalId` varchar(255) DEFAULT NULL,
  `resourceId` varchar(255) DEFAULT NULL,
  `resourceUpdatedAt` datetime(3) DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `schedule` varchar(100) DEFAULT NULL,
  `data` text DEFAULT NULL,
  `active` tinyint(1) DEFAULT NULL,
  `region` varchar(10) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_region_resourceType_resourceUpdatedAt` (`region`,`resourceType`,`resourceUpdatedAt`),
  KEY `_key_region_resourceType_projectId_resourceId` (`region`,`resourceType`,`projectId`,`resourceId`),
  KEY `_key_project_id_region` (`projectId`,`region`),
  KEY `_key_region_rt_active` (`region`,`resourceType`,`active`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_schedules`
--

LOCK TABLES `_console_schedules` WRITE;
/*!40000 ALTER TABLE `_console_schedules` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_schedules` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_schedules_perms`
--

DROP TABLE IF EXISTS `_console_schedules_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_schedules_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_schedules_perms`
--

LOCK TABLES `_console_schedules_perms` WRITE;
/*!40000 ALTER TABLE `_console_schedules_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_schedules_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_sessions`
--

DROP TABLE IF EXISTS `_console_sessions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_sessions` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `provider` varchar(128) DEFAULT NULL,
  `providerUid` varchar(2048) DEFAULT NULL,
  `providerAccessToken` text DEFAULT NULL,
  `providerAccessTokenExpiry` datetime(3) DEFAULT NULL,
  `providerRefreshToken` text DEFAULT NULL,
  `secret` varchar(512) DEFAULT NULL,
  `userAgent` text DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  `countryCode` varchar(2) DEFAULT NULL,
  `osCode` varchar(256) DEFAULT NULL,
  `osName` varchar(256) DEFAULT NULL,
  `osVersion` varchar(256) DEFAULT NULL,
  `clientType` varchar(256) DEFAULT NULL,
  `clientCode` varchar(256) DEFAULT NULL,
  `clientName` varchar(256) DEFAULT NULL,
  `clientVersion` varchar(256) DEFAULT NULL,
  `clientEngine` varchar(256) DEFAULT NULL,
  `clientEngineVersion` varchar(256) DEFAULT NULL,
  `deviceName` varchar(256) DEFAULT NULL,
  `deviceBrand` varchar(256) DEFAULT NULL,
  `deviceModel` varchar(256) DEFAULT NULL,
  `factors` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`factors`)),
  `expire` datetime(3) DEFAULT NULL,
  `mfaUpdatedAt` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_provider_providerUid` (`provider`,`providerUid`(128)),
  KEY `_key_user` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_sessions`
--

LOCK TABLES `_console_sessions` WRITE;
/*!40000 ALTER TABLE `_console_sessions` DISABLE KEYS */;
INSERT INTO `_console_sessions` VALUES
(1,'69ce38e7cb822b545d18','2026-04-02 09:37:43.889','2026-04-02 09:37:43.889','[\"read(\\\"user:69ce38e70026eaa5d8db\\\")\",\"update(\\\"user:69ce38e70026eaa5d8db\\\")\",\"delete(\\\"user:69ce38e70026eaa5d8db\\\")\"]','1','69ce38e70026eaa5d8db','email','admin@example.org',NULL,NULL,NULL,'{\"data\":\"LmX\\/EPqoJqNgoDk2CoUVlrq9mXShEyi13vpTQ5TfZ3iIuS9qk+K7PiGhafvmbYwxk\\/ORl2T4QZ6BCEnjisZmwg==\",\"method\":\"aes-128-gcm\",\"iv\":\"9a4c32cb8242d360a183439b\",\"tag\":\"ea35de1d0d1f1a5124b6aa2eb3c4b44d\",\"version\":\"1\"}','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36','172.28.0.1','--','WIN','Windows','10','browser','CH','Chrome','146.0','Blink','146.0.0.0','desktop',NULL,NULL,'[\"password\"]','2027-04-02 09:37:43.833',NULL);
/*!40000 ALTER TABLE `_console_sessions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_sessions_perms`
--

DROP TABLE IF EXISTS `_console_sessions_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_sessions_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_sessions_perms`
--

LOCK TABLES `_console_sessions_perms` WRITE;
/*!40000 ALTER TABLE `_console_sessions_perms` DISABLE KEYS */;
INSERT INTO `_console_sessions_perms` VALUES
(3,'delete','user:69ce38e70026eaa5d8db','69ce38e7cb822b545d18'),
(1,'read','user:69ce38e70026eaa5d8db','69ce38e7cb822b545d18'),
(2,'update','user:69ce38e70026eaa5d8db','69ce38e7cb822b545d18');
/*!40000 ALTER TABLE `_console_sessions_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_stats`
--

DROP TABLE IF EXISTS `_console_stats`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_stats` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `metric` varchar(255) DEFAULT NULL,
  `region` varchar(255) DEFAULT NULL,
  `value` bigint(20) DEFAULT NULL,
  `time` datetime(3) DEFAULT NULL,
  `period` varchar(4) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_metric_period_time` (`metric` DESC,`period`,`time`),
  KEY `_key_time` (`time` DESC),
  KEY `_key_period_time` (`period`,`time`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_stats`
--

LOCK TABLES `_console_stats` WRITE;
/*!40000 ALTER TABLE `_console_stats` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_stats` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_stats_perms`
--

DROP TABLE IF EXISTS `_console_stats_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_stats_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_stats_perms`
--

LOCK TABLES `_console_stats_perms` WRITE;
/*!40000 ALTER TABLE `_console_stats_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_stats_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_subscribers`
--

DROP TABLE IF EXISTS `_console_subscribers`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_subscribers` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `targetId` varchar(255) DEFAULT NULL,
  `targetInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `topicId` varchar(255) DEFAULT NULL,
  `topicInternalId` varchar(255) DEFAULT NULL,
  `providerType` varchar(128) DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_unique_target_topic` (`targetInternalId`,`topicInternalId`),
  KEY `_key_targetId` (`targetId`),
  KEY `_key_targetInternalId` (`targetInternalId`),
  KEY `_key_userId` (`userId`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_key_topicId` (`topicId`),
  KEY `_key_topicInternalId` (`topicInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_fulltext_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_subscribers`
--

LOCK TABLES `_console_subscribers` WRITE;
/*!40000 ALTER TABLE `_console_subscribers` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_subscribers` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_subscribers_perms`
--

DROP TABLE IF EXISTS `_console_subscribers_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_subscribers_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_subscribers_perms`
--

LOCK TABLES `_console_subscribers_perms` WRITE;
/*!40000 ALTER TABLE `_console_subscribers_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_subscribers_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_targets`
--

DROP TABLE IF EXISTS `_console_targets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_targets` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `sessionId` varchar(255) DEFAULT NULL,
  `sessionInternalId` varchar(255) DEFAULT NULL,
  `providerType` varchar(255) DEFAULT NULL,
  `providerId` varchar(255) DEFAULT NULL,
  `providerInternalId` varchar(255) DEFAULT NULL,
  `identifier` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `expired` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_identifier` (`identifier`),
  KEY `_key_userId` (`userId`),
  KEY `_key_userInternalId` (`userInternalId`),
  KEY `_key_providerId` (`providerId`),
  KEY `_key_providerInternalId` (`providerInternalId`),
  KEY `_key_expired` (`expired`),
  KEY `_key_session_internal_id` (`sessionInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_targets`
--

LOCK TABLES `_console_targets` WRITE;
/*!40000 ALTER TABLE `_console_targets` DISABLE KEYS */;
INSERT INTO `_console_targets` VALUES
(1,'69ce38e7bb42fd421be0','2026-04-02 09:37:43.767','2026-04-02 09:37:43.767','[\"read(\\\"user:69ce38e70026eaa5d8db\\\")\",\"update(\\\"user:69ce38e70026eaa5d8db\\\")\",\"delete(\\\"user:69ce38e70026eaa5d8db\\\")\"]','69ce38e70026eaa5d8db','1',NULL,NULL,'email',NULL,NULL,'admin@example.org',NULL,0);
/*!40000 ALTER TABLE `_console_targets` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_targets_perms`
--

DROP TABLE IF EXISTS `_console_targets_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_targets_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_targets_perms`
--

LOCK TABLES `_console_targets_perms` WRITE;
/*!40000 ALTER TABLE `_console_targets_perms` DISABLE KEYS */;
INSERT INTO `_console_targets_perms` VALUES
(3,'delete','user:69ce38e70026eaa5d8db','69ce38e7bb42fd421be0'),
(1,'read','user:69ce38e70026eaa5d8db','69ce38e7bb42fd421be0'),
(2,'update','user:69ce38e70026eaa5d8db','69ce38e7bb42fd421be0');
/*!40000 ALTER TABLE `_console_targets_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_teams`
--

DROP TABLE IF EXISTS `_console_teams`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_teams` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `total` int(11) DEFAULT NULL,
  `search` text DEFAULT NULL,
  `prefs` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_name` (`name`),
  KEY `_key_total` (`total`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_teams`
--

LOCK TABLES `_console_teams` WRITE;
/*!40000 ALTER TABLE `_console_teams` DISABLE KEYS */;
INSERT INTO `_console_teams` VALUES
(1,'69ce38ef00104ea230bc','2026-04-02 09:37:51.253','2026-04-02 09:37:51.253','[\"read(\\\"team:69ce38ef00104ea230bc\\\")\",\"update(\\\"team:69ce38ef00104ea230bc\\/owner\\\")\",\"delete(\\\"team:69ce38ef00104ea230bc\\/owner\\\")\"]','Personal projects',1,'69ce38ef00104ea230bc Personal projects','{}');
/*!40000 ALTER TABLE `_console_teams` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_teams_perms`
--

DROP TABLE IF EXISTS `_console_teams_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_teams_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_teams_perms`
--

LOCK TABLES `_console_teams_perms` WRITE;
/*!40000 ALTER TABLE `_console_teams_perms` DISABLE KEYS */;
INSERT INTO `_console_teams_perms` VALUES
(3,'delete','team:69ce38ef00104ea230bc/owner','69ce38ef00104ea230bc'),
(1,'read','team:69ce38ef00104ea230bc','69ce38ef00104ea230bc'),
(2,'update','team:69ce38ef00104ea230bc/owner','69ce38ef00104ea230bc');
/*!40000 ALTER TABLE `_console_teams_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_tokens`
--

DROP TABLE IF EXISTS `_console_tokens`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_tokens` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `userInternalId` varchar(255) DEFAULT NULL,
  `userId` varchar(255) DEFAULT NULL,
  `type` int(11) DEFAULT NULL,
  `secret` varchar(512) DEFAULT NULL,
  `expire` datetime(3) DEFAULT NULL,
  `userAgent` text DEFAULT NULL,
  `ip` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_user` (`userInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_tokens`
--

LOCK TABLES `_console_tokens` WRITE;
/*!40000 ALTER TABLE `_console_tokens` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_tokens` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_tokens_perms`
--

DROP TABLE IF EXISTS `_console_tokens_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_tokens_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_tokens_perms`
--

LOCK TABLES `_console_tokens_perms` WRITE;
/*!40000 ALTER TABLE `_console_tokens_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_tokens_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_topics`
--

DROP TABLE IF EXISTS `_console_topics`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_topics` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(128) DEFAULT NULL,
  `subscribe` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`subscribe`)),
  `emailTotal` int(11) DEFAULT NULL,
  `smsTotal` int(11) DEFAULT NULL,
  `pushTotal` int(11) DEFAULT NULL,
  `targets` text DEFAULT NULL,
  `search` text DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_name` (`name`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_topics`
--

LOCK TABLES `_console_topics` WRITE;
/*!40000 ALTER TABLE `_console_topics` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_topics` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_topics_perms`
--

DROP TABLE IF EXISTS `_console_topics_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_topics_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_topics_perms`
--

LOCK TABLES `_console_topics_perms` WRITE;
/*!40000 ALTER TABLE `_console_topics_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_topics_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_users`
--

DROP TABLE IF EXISTS `_console_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_users` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `email` varchar(320) DEFAULT NULL,
  `phone` varchar(16) DEFAULT NULL,
  `status` tinyint(1) DEFAULT NULL,
  `labels` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`labels`)),
  `passwordHistory` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`passwordHistory`)),
  `password` text DEFAULT NULL,
  `hash` varchar(256) DEFAULT NULL,
  `hashOptions` text DEFAULT NULL,
  `passwordUpdate` datetime(3) DEFAULT NULL,
  `prefs` text DEFAULT NULL,
  `registration` datetime(3) DEFAULT NULL,
  `emailVerification` tinyint(1) DEFAULT NULL,
  `phoneVerification` tinyint(1) DEFAULT NULL,
  `reset` tinyint(1) DEFAULT NULL,
  `mfa` tinyint(1) DEFAULT NULL,
  `mfaRecoveryCodes` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`mfaRecoveryCodes`)),
  `authenticators` text DEFAULT NULL,
  `sessions` text DEFAULT NULL,
  `tokens` text DEFAULT NULL,
  `challenges` text DEFAULT NULL,
  `memberships` text DEFAULT NULL,
  `targets` text DEFAULT NULL,
  `search` text DEFAULT NULL,
  `accessedAt` datetime(3) DEFAULT NULL,
  `emailCanonical` varchar(320) DEFAULT NULL,
  `emailIsFree` tinyint(1) DEFAULT NULL,
  `emailIsDisposable` tinyint(1) DEFAULT NULL,
  `emailIsCorporate` tinyint(1) DEFAULT NULL,
  `emailIsCanonical` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  UNIQUE KEY `_key_phone` (`phone`),
  UNIQUE KEY `_key_email` (`email`(256)),
  KEY `_key_name` (`name`),
  KEY `_key_status` (`status`),
  KEY `_key_passwordUpdate` (`passwordUpdate`),
  KEY `_key_registration` (`registration`),
  KEY `_key_emailVerification` (`emailVerification`),
  KEY `_key_phoneVerification` (`phoneVerification`),
  KEY `_key_accessedAt` (`accessedAt`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`),
  FULLTEXT KEY `_key_search` (`search`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_users`
--

LOCK TABLES `_console_users` WRITE;
/*!40000 ALTER TABLE `_console_users` DISABLE KEYS */;
INSERT INTO `_console_users` VALUES
(1,'69ce38e70026eaa5d8db','2026-04-02 09:37:43.737','2026-04-02 09:51:29.422','[\"read(\\\"any\\\")\",\"update(\\\"user:69ce38e70026eaa5d8db\\\")\",\"delete(\\\"user:69ce38e70026eaa5d8db\\\")\"]','dev admin','admin@example.org',NULL,1,'[]','[]','{\"data\":\"OfIXDk7B+OKqRll1th79iTG0v5nuhDIE6vw1vzq+0SPgcrOw4vCqrQ7Dr68CoDxXroDt4RyLRaKzNphbj8DC3UHJ2aSJosbdzqg4755RcEYHniLVPn9bV1g92hrZiEwGvg==\",\"method\":\"aes-128-gcm\",\"iv\":\"b2503ececf43de43430f8c2f\",\"tag\":\"09b73c0ee88ff25004f07f18d8d7303d\",\"version\":\"1\"}','argon2','{\"type\":\"argon2\",\"memory_cost\":65536,\"time_cost\":4,\"threads\":3}','2026-04-02 09:37:43.724','{\"organization\":\"69ce38ef00104ea230bc\",\"newOnboardingCompleted\":true,\"console\":{\"\\/(console)\\/project-[region]-[project]\\/storage\":{\"view\":\"table\"},\"\\/(console)\\/project-[region]-[project]\\/databases\":{\"view\":\"table\"}}}','2026-04-02 09:37:43.724',0,NULL,0,0,'[]',NULL,NULL,NULL,NULL,NULL,NULL,'69ce38e70026eaa5d8db admin@example.org dev admin','2026-04-02 09:37:43.724','admin@example.org',0,0,1,0);
/*!40000 ALTER TABLE `_console_users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_users_perms`
--

DROP TABLE IF EXISTS `_console_users_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_users_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_users_perms`
--

LOCK TABLES `_console_users_perms` WRITE;
/*!40000 ALTER TABLE `_console_users_perms` DISABLE KEYS */;
INSERT INTO `_console_users_perms` VALUES
(3,'delete','user:69ce38e70026eaa5d8db','69ce38e70026eaa5d8db'),
(1,'read','any','69ce38e70026eaa5d8db'),
(2,'update','user:69ce38e70026eaa5d8db','69ce38e70026eaa5d8db');
/*!40000 ALTER TABLE `_console_users_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_vcsCommentLocks`
--

DROP TABLE IF EXISTS `_console_vcsCommentLocks`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_vcsCommentLocks` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_vcsCommentLocks`
--

LOCK TABLES `_console_vcsCommentLocks` WRITE;
/*!40000 ALTER TABLE `_console_vcsCommentLocks` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_vcsCommentLocks` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_vcsCommentLocks_perms`
--

DROP TABLE IF EXISTS `_console_vcsCommentLocks_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_vcsCommentLocks_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_vcsCommentLocks_perms`
--

LOCK TABLES `_console_vcsCommentLocks_perms` WRITE;
/*!40000 ALTER TABLE `_console_vcsCommentLocks_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_vcsCommentLocks_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_vcsComments`
--

DROP TABLE IF EXISTS `_console_vcsComments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_vcsComments` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `installationId` varchar(255) DEFAULT NULL,
  `installationInternalId` varchar(255) DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `providerRepositoryId` varchar(255) DEFAULT NULL,
  `providerCommentId` varchar(255) DEFAULT NULL,
  `providerPullRequestId` varchar(255) DEFAULT NULL,
  `providerBranch` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_installationId` (`installationId`),
  KEY `_key_installationInternalId` (`installationInternalId`),
  KEY `_key_projectInternalId` (`projectInternalId`),
  KEY `_key_projectId` (`projectId`),
  KEY `_key_providerRepositoryId` (`providerRepositoryId`),
  KEY `_key_providerPullRequestId` (`providerPullRequestId`),
  KEY `_key_providerBranch` (`providerBranch`),
  KEY `_key_piid_prid_rt` (`projectInternalId`,`providerRepositoryId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_vcsComments`
--

LOCK TABLES `_console_vcsComments` WRITE;
/*!40000 ALTER TABLE `_console_vcsComments` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_vcsComments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_vcsComments_perms`
--

DROP TABLE IF EXISTS `_console_vcsComments_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_vcsComments_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_vcsComments_perms`
--

LOCK TABLES `_console_vcsComments_perms` WRITE;
/*!40000 ALTER TABLE `_console_vcsComments_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_vcsComments_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_webhooks`
--

DROP TABLE IF EXISTS `_console_webhooks`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_webhooks` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `projectInternalId` varchar(255) DEFAULT NULL,
  `projectId` varchar(255) DEFAULT NULL,
  `name` varchar(255) DEFAULT NULL,
  `url` varchar(255) DEFAULT NULL,
  `httpUser` varchar(255) DEFAULT NULL,
  `httpPass` varchar(255) DEFAULT NULL,
  `security` tinyint(1) DEFAULT NULL,
  `events` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`events`)),
  `signatureKey` varchar(2048) DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `logs` mediumtext DEFAULT NULL,
  `attempts` int(11) DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`),
  KEY `_key_project` (`projectInternalId`),
  KEY `_created_at` (`_createdAt`),
  KEY `_updated_at` (`_updatedAt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_webhooks`
--

LOCK TABLES `_console_webhooks` WRITE;
/*!40000 ALTER TABLE `_console_webhooks` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_webhooks` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `_console_webhooks_perms`
--

DROP TABLE IF EXISTS `_console_webhooks_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `_console_webhooks_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_type`,`_permission`),
  KEY `_permission` (`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `_console_webhooks_perms`
--

LOCK TABLES `_console_webhooks_perms` WRITE;
/*!40000 ALTER TABLE `_console_webhooks_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `_console_webhooks_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `logsV1__metadata`
--

DROP TABLE IF EXISTS `logsV1__metadata`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `logsV1__metadata` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `name` varchar(256) DEFAULT NULL,
  `attributes` mediumtext DEFAULT NULL,
  `indexes` mediumtext DEFAULT NULL,
  `documentSecurity` tinyint(1) DEFAULT NULL,
  `_tenant` int(11) unsigned DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_uid` (`_uid`,`_tenant`),
  KEY `_created_at` (`_tenant`,`_createdAt`),
  KEY `_updated_at` (`_tenant`,`_updatedAt`),
  KEY `_tenant_id` (`_tenant`,`_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `logsV1__metadata`
--

LOCK TABLES `logsV1__metadata` WRITE;
/*!40000 ALTER TABLE `logsV1__metadata` DISABLE KEYS */;
INSERT INTO `logsV1__metadata` VALUES
(1,'stats','2026-04-02 09:37:24.905','2026-04-02 09:37:24.905','[\"create(\\\"any\\\")\"]','stats','[{\"$id\":\"metric\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"region\",\"type\":\"string\",\"size\":255,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"value\",\"type\":\"integer\",\"size\":8,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"},{\"$id\":\"time\",\"type\":\"datetime\",\"size\":0,\"required\":false,\"signed\":false,\"array\":false,\"filters\":[\"datetime\"],\"default\":null,\"format\":\"\"},{\"$id\":\"period\",\"type\":\"string\",\"size\":4,\"required\":true,\"signed\":true,\"array\":false,\"filters\":[],\"default\":null,\"format\":\"\"}]','[{\"$id\":\"_key_time\",\"type\":\"key\",\"attributes\":[\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]},{\"$id\":\"_key_period_time\",\"type\":\"key\",\"attributes\":[\"period\",\"time\"],\"lengths\":[],\"orders\":[\"ASC\"]},{\"$id\":\"_key_metric_period_time\",\"type\":\"unique\",\"attributes\":[\"metric\",\"period\",\"time\"],\"lengths\":[],\"orders\":[\"DESC\"]}]',1,NULL);
/*!40000 ALTER TABLE `logsV1__metadata` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `logsV1__metadata_perms`
--

DROP TABLE IF EXISTS `logsV1__metadata_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `logsV1__metadata_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  `_tenant` int(11) unsigned DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_tenant`,`_type`,`_permission`),
  KEY `_permission` (`_tenant`,`_permission`,`_type`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `logsV1__metadata_perms`
--

LOCK TABLES `logsV1__metadata_perms` WRITE;
/*!40000 ALTER TABLE `logsV1__metadata_perms` DISABLE KEYS */;
INSERT INTO `logsV1__metadata_perms` VALUES
(1,'create','any','stats',NULL);
/*!40000 ALTER TABLE `logsV1__metadata_perms` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `logsV1_stats`
--

DROP TABLE IF EXISTS `logsV1_stats`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `logsV1_stats` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_uid` varchar(255) NOT NULL,
  `_createdAt` datetime(3) DEFAULT NULL,
  `_updatedAt` datetime(3) DEFAULT NULL,
  `_permissions` mediumtext DEFAULT NULL,
  `metric` varchar(255) DEFAULT NULL,
  `region` varchar(255) DEFAULT NULL,
  `value` bigint(20) DEFAULT NULL,
  `time` datetime(3) DEFAULT NULL,
  `period` varchar(4) DEFAULT NULL,
  `_tenant` int(11) unsigned DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_key_metric_period_time` (`_tenant`,`metric` DESC,`period`,`time`),
  UNIQUE KEY `_uid` (`_uid`,`_tenant`),
  KEY `_key_time` (`_tenant`,`time` DESC),
  KEY `_key_period_time` (`_tenant`,`period`,`time`),
  KEY `_created_at` (`_tenant`,`_createdAt`),
  KEY `_updated_at` (`_tenant`,`_updatedAt`),
  KEY `_tenant_id` (`_tenant`,`_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `logsV1_stats`
--

LOCK TABLES `logsV1_stats` WRITE;
/*!40000 ALTER TABLE `logsV1_stats` DISABLE KEYS */;
/*!40000 ALTER TABLE `logsV1_stats` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `logsV1_stats_perms`
--

DROP TABLE IF EXISTS `logsV1_stats_perms`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `logsV1_stats_perms` (
  `_id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `_type` varchar(12) NOT NULL,
  `_permission` varchar(255) NOT NULL,
  `_document` varchar(255) NOT NULL,
  `_tenant` int(11) unsigned DEFAULT NULL,
  PRIMARY KEY (`_id`),
  UNIQUE KEY `_index1` (`_document`,`_tenant`,`_type`,`_permission`),
  KEY `_permission` (`_tenant`,`_permission`,`_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `logsV1_stats_perms`
--

LOCK TABLES `logsV1_stats_perms` WRITE;
/*!40000 ALTER TABLE `logsV1_stats_perms` DISABLE KEYS */;
/*!40000 ALTER TABLE `logsV1_stats_perms` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-04-02  9:57:44
