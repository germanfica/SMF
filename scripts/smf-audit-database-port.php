<?php

// Read-only exhaustive audit program for a running SMF container. It reads
// every row and every column of every table in the connected SMF database and
// reports values containing the exact TCP port supplied by SMF_AUDIT_PORT.
// It intentionally performs no UPDATE, INSERT, DELETE, DDL, or transaction.
//
// This can be slow on a large forum and may report matches in user-generated
// content. Output contains a bounded excerpt rather than an entire post or BLOB.

function smfDatabaseAuditFail($message)
{
	fwrite(STDERR, $message . "\n");
	exit(1);
}

function smfDatabaseAuditQueryOrFail($query, $parameters, $message)
{
	global $smcFunc;

	$parameters['db_error_skip'] = true;
	$result = $smcFunc['db_query']('', $query, $parameters);
	if ($result === false) {
		smfDatabaseAuditFail($message);
	}

	return $result;
}

function smfDatabaseAuditQuoteIdentifier($identifier)
{
	return '`' . str_replace('`', '``', $identifier) . '`';
}

function smfDatabaseAuditPortPattern($port)
{
	return '/(?<![0-9]):' . preg_quote((string) $port, '/') . '(?![0-9])/';
}

function smfDatabaseAuditExcerpt($value, $port)
{
	if (!is_string($value) || preg_match(smfDatabaseAuditPortPattern($port), $value, $match, PREG_OFFSET_CAPTURE) !== 1) {
		return null;
	}

	$matchOffset = $match[0][1];
	$prefixLength = 240;
	$suffixLength = 1024;
	$startOffset = max(0, $matchOffset - $prefixLength);
	$endOffset = min(strlen($value), $matchOffset + strlen($match[0][0]) + $suffixLength);
	$excerpt = substr($value, $startOffset, $endOffset - $startOffset);
	if ($startOffset > 0) {
		$excerpt = '...' . $excerpt;
	}
	if ($endOffset < strlen($value)) {
		$excerpt .= '...';
	}

	return $excerpt;
}

function smfDatabaseAuditRedactSensitiveValues($excerpt)
{
	$sensitiveName = '(?:webmaster_email|email|ip2?|login_[A-Za-z0-9_-]+|[A-Za-z_][A-Za-z0-9_-]*(?:password|passwd|secret|token|api[_-]?key|private[_-]?key|client[_-]?secret|access[_-]?key|webhook|cookie)[A-Za-z0-9_-]*)';
	$excerpt = preg_replace_callback(
		'#(\$' . $sensitiveName . '\s*=\s*)([\'\"])(?:\\\\.|(?!\2).)*\2#is',
		function ($match) {
			return $match[1] . $match[2] . '****' . $match[2];
		},
		$excerpt
	);
	$excerpt = preg_replace_callback(
		'#(^[ \t]*(?:export[ \t]+)?' . $sensitiveName . '[ \t]*=[ \t]*)([\'\"]?)[^\r\n]*\2#im',
		function ($match) {
			return $match[1] . ($match[2] === '' ? '' : $match[2]) . '****' . ($match[2] === '' ? '' : $match[2]);
		},
		$excerpt
	);
	$excerpt = preg_replace_callback(
		'#((?:[\'\"]?' . $sensitiveName . '[\'\"]?)[ \t]*:[ \t]*)([\'\"])(?:\\\\.|(?!\2).)*\2#is',
		function ($match) {
			return $match[1] . $match[2] . '****' . $match[2];
		},
		$excerpt
	);

	$excerpt = smfDatabaseAuditRedactSerializedSensitiveValues($excerpt, $sensitiveName);

	return smfDatabaseAuditRedactIpAddresses($excerpt);
}

function smfDatabaseAuditRedactSerializedSensitiveValues($excerpt, $sensitiveName)
{
	$pattern = '#s:\d+:"' . $sensitiveName . '";s:(\d+):"#i';
	$searchOffset = 0;
	while (preg_match($pattern, $excerpt, $match, PREG_OFFSET_CAPTURE, $searchOffset) === 1) {
		$prefix = $match[0][0];
		$prefixOffset = $match[0][1];
		$valueLength = (int) $match[1][0];
		$valueOffset = $prefixOffset + strlen($prefix);
		$valueEndOffset = $valueOffset + $valueLength;
		if ($valueEndOffset >= strlen($excerpt) || substr($excerpt, $valueEndOffset, 2) !== '";') {
			$searchOffset = $valueOffset;
			continue;
		}

		$excerpt = substr($excerpt, 0, $valueOffset) . '****' . substr($excerpt, $valueEndOffset);
		$searchOffset = $valueOffset + 6;
	}

	return $excerpt;
}

function smfDatabaseAuditRedactIpAddresses($excerpt)
{
	$excerpt = preg_replace(
		'/(?<![0-9])(?:25[0-5]|2[0-4][0-9]|1?[0-9]{1,2})(?:\.(?:25[0-5]|2[0-4][0-9]|1?[0-9]{1,2})){3}(?![0-9])/',
		'****',
		$excerpt
	);

	return preg_replace_callback(
		'/(?<![0-9A-Fa-f:.])[0-9A-Fa-f:.]{2,45}(?![0-9A-Fa-f:.])/',
		function ($match) {
			return filter_var($match[0], FILTER_VALIDATE_IP, FILTER_FLAG_IPV6) !== false ? '****' : $match[0];
		},
		$excerpt
	);
}

function smfDatabaseAuditRedactRowIdentity($rowIdentity)
{
	foreach ($rowIdentity as $name => $value) {
		if ((is_string($value) && filter_var($value, FILTER_VALIDATE_IP) !== false) || preg_match('/(?:session|password|passwd|secret|token|api[_-]?key|private[_-]?key|client[_-]?secret|access[_-]?key|webhook|cookie)/i', $name)) {
			$rowIdentity[$name] = '****';
		}
	}

	return $rowIdentity;
}

function smfDatabaseAuditPrimaryKeyColumns($tableName)
{
	global $smcFunc;

	$result = smfDatabaseAuditQueryOrFail(
		'SHOW KEYS FROM {raw:table_name} WHERE Key_name = {string:key_name}',
		array(
			'table_name' => smfDatabaseAuditQuoteIdentifier($tableName),
			'key_name' => 'PRIMARY',
		),
		'Could not read primary-key metadata for table ' . $tableName . '.'
	);
	$columns = array();
	while ($row = $smcFunc['db_fetch_assoc']($result)) {
		$columns[(int) $row['Seq_in_index']] = $row['Column_name'];
	}
	$smcFunc['db_free_result']($result);
	ksort($columns);

	return array_values($columns);
}

function smfDatabaseAuditRowIdentity($row, $primaryKeyColumns, $rowNumber)
{
	if (empty($primaryKeyColumns)) {
		return array('_row' => $rowNumber);
	}

	$identity = array();
	foreach ($primaryKeyColumns as $columnName) {
		$identity[$columnName] = array_key_exists($columnName, $row) ? $row[$columnName] : null;
	}

	return $identity;
}

function smfDatabaseAuditPrintMatch($tableName, $rowIdentity, $columnName, $excerpt)
{
	global $smfDatabaseAuditMatchCount;

	$smfDatabaseAuditMatchCount++;
	fwrite(
		STDOUT,
		$tableName . "\t" .
		json_encode(smfDatabaseAuditRedactRowIdentity($rowIdentity), JSON_UNESCAPED_SLASHES | JSON_INVALID_UTF8_SUBSTITUTE) . "\t" .
		$columnName . "\t" .
		json_encode(smfDatabaseAuditRedactSensitiveValues($excerpt), JSON_UNESCAPED_SLASHES | JSON_INVALID_UTF8_SUBSTITUTE) . "\n"
	);
}

$port = getenv('SMF_AUDIT_PORT');
if (!is_string($port) || !preg_match('/^[0-9]+$/', $port) || (int) $port < 1 || (int) $port > 65535) {
	smfDatabaseAuditFail('SMF_AUDIT_PORT must be a TCP port from 1 through 65535.');
}
$port = (int) $port;

$settingsPath = realpath('/var/www/html/Settings.php');
if ($settingsPath === false || dirname($settingsPath) !== '/var/www/smf-config' || !is_file($settingsPath)) {
	smfDatabaseAuditFail('Could not resolve the persisted Settings.php target.');
}

define('SMF', 1);
define('SMF_VERSION', '2.1.7');
define('SMF_FULL_VERSION', 'SMF ' . SMF_VERSION);
define('SMF_SOFTWARE_YEAR', '2026');
define('POSTGRE_TITLE', 'PostgreSQL');
define('MYSQL_TITLE', 'MySQL');
define('SMF_USER_AGENT', 'SMF database port audit');
define('TIME_START', microtime(true));

require_once($settingsPath);
foreach (array('db_character_set', 'cachedir') as $variable) {
	unset($GLOBALS[$variable]);
}
foreach (array('QueryString.php', 'Subs.php', 'Subs-Auth.php', 'Errors.php', 'Load.php', 'Security.php') as $sourceFile) {
	$sourcePath = $sourcedir . '/' . $sourceFile;
	if (!is_file($sourcePath)) {
		smfDatabaseAuditFail('The SMF source file ' . $sourceFile . ' is unavailable.');
	}
	require_once($sourcePath);
}
if (version_compare(PHP_VERSION, '8.0.0', '>=')) {
	require_once($sourcedir . '/Subs-Compat.php');
}

$smcFunc = array();
loadDatabase();

$tablesResult = smfDatabaseAuditQueryOrFail('SHOW TABLES', array(), 'Could not list SMF database tables.');
$tableNames = array();
while ($tableRow = $smcFunc['db_fetch_row']($tablesResult)) {
	$tableNames[] = $tableRow[0];
}
$smcFunc['db_free_result']($tablesResult);

$smfDatabaseAuditMatchCount = 0;
$smfDatabaseAuditRowCount = 0;
fwrite(STDOUT, "table\tprimary_key\tcolumn\texcerpt\n");
foreach ($tableNames as $tableName) {
	$primaryKeyColumns = smfDatabaseAuditPrimaryKeyColumns($tableName);
	$result = smfDatabaseAuditQueryOrFail(
		'SELECT * FROM {raw:table_name}',
		array('table_name' => smfDatabaseAuditQuoteIdentifier($tableName)),
		'Could not read table ' . $tableName . '.'
	);
	$tableRowNumber = 0;
	while ($row = $smcFunc['db_fetch_assoc']($result)) {
		$tableRowNumber++;
		$smfDatabaseAuditRowCount++;
		$rowIdentity = smfDatabaseAuditRowIdentity($row, $primaryKeyColumns, $tableRowNumber);
		foreach ($row as $columnName => $value) {
			$excerpt = smfDatabaseAuditExcerpt($value, $port);
			if ($excerpt !== null) {
				smfDatabaseAuditPrintMatch($tableName, $rowIdentity, $columnName, $excerpt);
			}
		}
	}
	$smcFunc['db_free_result']($result);
}

fwrite(
	STDOUT,
	"Scanned " . count($tableNames) . " tables and " . $smfDatabaseAuditRowCount .
	" rows; found " . $smfDatabaseAuditMatchCount . " values containing TCP port " . $port . ".\n"
);
