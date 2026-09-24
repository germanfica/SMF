<?php

// Read-only audit program for a running SMF container. It reports Settings.php,
// settings, and global theme values that contain the exact TCP port supplied by
// SMF_AUDIT_PORT. It intentionally performs no UPDATE, INSERT, or DELETE.

function smfAuditFail($message)
{
	fwrite(STDERR, $message . "\n");
	exit(1);
}

function smfAuditValueContainsPort($value, $port)
{
	return is_string($value) && preg_match('/(?<![0-9]):' . preg_quote((string) $port, '/') . '(?![0-9])/', $value) === 1;
}

function smfAuditPrintMatch($location, $value)
{
	global $smfAuditMatches;
	$smfAuditMatches++;
	fwrite(STDOUT, $location . "\t" . json_encode($value, JSON_UNESCAPED_SLASHES) . "\n");
}

$smfAuditMatches = 0;
$port = getenv('SMF_AUDIT_PORT');
if (!is_string($port) || !preg_match('/^[0-9]+$/', $port) || (int) $port < 1 || (int) $port > 65535) {
	smfAuditFail('SMF_AUDIT_PORT must be a TCP port from 1 through 65535.');
}
$port = (int) $port;

$settingsPath = realpath('/var/www/html/Settings.php');
if ($settingsPath === false || dirname($settingsPath) !== '/var/www/smf-config' || !is_file($settingsPath)) {
	smfAuditFail('Could not resolve the persisted Settings.php target.');
}

define('SMF', 1);
define('SMF_VERSION', '2.1.7');
define('SMF_FULL_VERSION', 'SMF ' . SMF_VERSION);
define('SMF_SOFTWARE_YEAR', '2026');
define('POSTGRE_TITLE', 'PostgreSQL');
define('MYSQL_TITLE', 'MySQL');
define('SMF_USER_AGENT', 'SMF URL port audit');
define('TIME_START', microtime(true));

require_once($settingsPath);
foreach (array('db_character_set', 'cachedir') as $variable) {
	unset($GLOBALS[$variable]);
}
foreach (array('QueryString.php', 'Subs.php', 'Subs-Auth.php', 'Errors.php', 'Load.php', 'Security.php') as $sourceFile) {
	$sourcePath = $sourcedir . '/' . $sourceFile;
	if (!is_file($sourcePath)) {
		smfAuditFail('The SMF source file ' . $sourceFile . ' is unavailable.');
	}
	require_once($sourcePath);
}
if (version_compare(PHP_VERSION, '8.0.0', '>=')) {
	require_once($sourcedir . '/Subs-Compat.php');
}

if (smfAuditValueContainsPort($boardurl, $port)) {
	smfAuditPrintMatch('Settings.php:$boardurl', $boardurl);
}

$smcFunc = array();
loadDatabase();

$result = $smcFunc['db_query']('', '
	SELECT variable, value
	FROM {db_prefix}settings', array(
	'db_error_skip' => true,
));
if ($result === false) {
	smfAuditFail('Could not read the SMF settings table.');
}
while ($row = $smcFunc['db_fetch_assoc']($result)) {
	if (smfAuditValueContainsPort($row['value'], $port)) {
		smfAuditPrintMatch('settings:' . $row['variable'], $row['value']);
	}
}
$smcFunc['db_free_result']($result);

$result = $smcFunc['db_query']('', '
	SELECT id_theme, variable, value
	FROM {db_prefix}themes
	WHERE id_member = {int:global_member}', array(
	'global_member' => 0,
	'db_error_skip' => true,
));
if ($result === false) {
	smfAuditFail('Could not read the global SMF theme settings.');
}
while ($row = $smcFunc['db_fetch_assoc']($result)) {
	if (smfAuditValueContainsPort($row['value'], $port)) {
		smfAuditPrintMatch('themes:' . $row['id_theme'] . ':' . $row['variable'], $row['value']);
	}
}
$smcFunc['db_free_result']($result);

if ($smfAuditMatches === 0) {
	fwrite(STDOUT, "No configuration values contain TCP port " . $port . ".\n");
}
