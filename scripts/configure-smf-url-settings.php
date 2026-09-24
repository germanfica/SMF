<?php

// This program is evaluated inside the running SMF container by
// playbooks/configure-smf.yml. Keep it independent from HTTP input: the
// requested public URL arrives only through SMF_FORUM_URL.

function smfConfigureURLFail($message)
{
	fwrite(STDERR, $message . "\n");
	exit(1);
}

function smfConfigureURLParts($url)
{
	$parts = parse_url($url);
	if (
		$parts === false ||
		!isset($parts['scheme']) ||
		!isset($parts['host']) ||
		($parts['scheme'] !== 'http' && $parts['scheme'] !== 'https') ||
		isset($parts['user']) ||
		isset($parts['pass']) ||
		isset($parts['query']) ||
		isset($parts['fragment'])
	) {
		return null;
	}

	return $parts;
}

function smfConfigureURLPath($parts)
{
	if (!isset($parts['path']) || $parts['path'] === '/') {
		return '';
	}

	return rtrim($parts['path'], '/');
}

function smfConfigureSameSchemeAndHost($leftParts, $rightParts)
{
	return strtolower($leftParts['scheme']) === strtolower($rightParts['scheme']) &&
		strtolower($leftParts['host']) === strtolower($rightParts['host']);
}

function smfConfigureRecoveryPathMatches($variable, $valuePath, $targetPath)
{
	$expectedPaths = array(
		'avatar_url' => '/avatars',
		'custom_avatar_url' => '/custom_avatar',
		'smileys_url' => '/Smileys',
	);
	if (isset($expectedPaths[$variable])) {
		return $valuePath === $targetPath . $expectedPaths[$variable];
	}

	return strpos($valuePath, $targetPath . '/Themes/') === 0;
}

function smfConfigureRewriteURL($value, $previousForumURL, $targetForumURL, $variable)
{
	if (!is_string($value) || $value === '') {
		return null;
	}

	$previousPrefix = $previousForumURL . '/';
	if (strpos($value, $previousPrefix) === 0) {
		return $targetForumURL . substr($value, strlen($previousForumURL));
	}

	// A prior configure invocation may already have updated $boardurl while
	// leaving generated asset URLs on the previous host port. Recover only
	// standard local SMF paths on the same scheme and host. External CDNs use
	// another host and are deliberately left unchanged.
	if ($previousForumURL !== $targetForumURL) {
		return null;
	}

	$valueParts = smfConfigureURLParts($value);
	$targetParts = smfConfigureURLParts($targetForumURL);
	if ($valueParts === null || $targetParts === null || !smfConfigureSameSchemeAndHost($valueParts, $targetParts)) {
		return null;
	}

	$targetPath = smfConfigureURLPath($targetParts);
	$valuePath = smfConfigureURLPath($valueParts);
	if (!smfConfigureRecoveryPathMatches($variable, $valuePath, $targetPath)) {
		return null;
	}

	return $targetForumURL . substr($valuePath, strlen($targetPath));
}

function smfConfigureQueryOrFail($query, $parameters, $message)
{
	global $smcFunc;

	$parameters['db_error_skip'] = true;
	$result = $smcFunc['db_query']('', $query, $parameters);
	if ($result === false) {
		throw new RuntimeException($message);
	}

	return $result;
}

function smfConfigureVerifySetting($change)
{
	global $smcFunc;

	$result = smfConfigureQueryOrFail('
		SELECT value
		FROM {db_prefix}settings
		WHERE variable = {string:variable}
		LIMIT 1', array(
		'variable' => $change['variable'],
	), 'Could not verify the SMF setting ' . $change['variable'] . '.');
	$row = $smcFunc['db_fetch_row']($result);
	$smcFunc['db_free_result']($result);
	if ($row === false || $row[0] !== $change['replacement']) {
		throw new RuntimeException('SMF setting ' . $change['variable'] . ' does not contain the requested URL.');
	}
}

function smfConfigureVerifyThemeSetting($change)
{
	global $smcFunc;

	$result = smfConfigureQueryOrFail('
		SELECT value
		FROM {db_prefix}themes
		WHERE id_theme = {int:id_theme}
			AND id_member = {int:global_member}
			AND variable = {string:variable}
		LIMIT 1', array(
		'id_theme' => $change['id_theme'],
		'global_member' => 0,
		'variable' => $change['variable'],
	), 'Could not verify the SMF theme URL setting.');
	$row = $smcFunc['db_fetch_row']($result);
	$smcFunc['db_free_result']($result);
	if ($row === false || $row[0] !== $change['replacement']) {
		throw new RuntimeException('An SMF theme URL setting does not contain the requested URL.');
	}
}

$settingsPath = realpath('/var/www/html/Settings.php');
$forumURL = getenv('SMF_FORUM_URL');
if (
	$settingsPath === false ||
	dirname($settingsPath) !== '/var/www/smf-config' ||
	!is_file($settingsPath) ||
	!is_writable($settingsPath) ||
	!is_writable(dirname($settingsPath))
) {
	smfConfigureURLFail('The persisted Settings.php target is not writable by the SMF service.');
}
if (!is_string($forumURL) || $forumURL === '' || smfConfigureURLParts($forumURL) === null || substr($forumURL, -1) === '/') {
	smfConfigureURLFail('SMF_FORUM_URL must be an http or https URL without a trailing slash.');
}

$settingsLines = file($settingsPath);
$boardURLLines = 0;
foreach ($settingsLines as $settingsLine) {
	if (preg_match('/^\s*\$boardurl\s*=/', $settingsLine)) {
		$boardURLLines++;
	}
}
if ($boardURLLines !== 1) {
	smfConfigureURLFail('Settings.php must contain exactly one $boardurl assignment.');
}

define('SMF', 1);
define('SMF_VERSION', '2.1.7');
define('SMF_FULL_VERSION', 'SMF ' . SMF_VERSION);
define('SMF_SOFTWARE_YEAR', '2026');
define('POSTGRE_TITLE', 'PostgreSQL');
define('MYSQL_TITLE', 'MySQL');
define('SMF_USER_AGENT', 'SMF configure');
define('TIME_START', microtime(true));

require_once($settingsPath);
$previousForumURL = rtrim($boardurl, '/');
if ($previousForumURL === '' || smfConfigureURLParts($previousForumURL) === null) {
	smfConfigureURLFail('Settings.php does not define a valid $boardurl.');
}

foreach (array('db_character_set', 'cachedir') as $variable) {
	unset($GLOBALS[$variable]);
}
foreach (array('QueryString.php', 'Subs.php', 'Subs-Auth.php', 'Errors.php', 'Load.php', 'Security.php') as $sourceFile) {
	$sourcePath = $sourcedir . '/' . $sourceFile;
	if (!is_file($sourcePath)) {
		smfConfigureURLFail('The SMF source file ' . $sourceFile . ' is unavailable.');
	}
	require_once($sourcePath);
}
if (version_compare(PHP_VERSION, '8.0.0', '>=')) {
	require_once($sourcedir . '/Subs-Compat.php');
}

$smcFunc = array();
loadDatabase();
$context = array();
reloadSettings();
require_once($sourcedir . '/Subs-Admin.php');

$settingChanges = array();
foreach (array('avatar_url', 'custom_avatar_url', 'smileys_url') as $variable) {
	if (!array_key_exists($variable, $modSettings)) {
		continue;
	}
	$replacement = smfConfigureRewriteURL($modSettings[$variable], $previousForumURL, $forumURL, $variable);
	if ($replacement !== null && $replacement !== $modSettings[$variable]) {
		$settingChanges[] = array(
			'variable' => $variable,
			'previous' => $modSettings[$variable],
			'replacement' => $replacement,
		);
	}
}

$themeChanges = array();
$themeVariables = array('theme_url', 'images_url', 'base_theme_url', 'base_images_url');
$themeResult = smfConfigureQueryOrFail('
	SELECT id_theme, variable, value
	FROM {db_prefix}themes
	WHERE id_member = {int:global_member}
		AND variable IN ({array_string:variables})', array(
	'global_member' => 0,
	'variables' => $themeVariables,
), 'Could not read the global SMF theme URL settings.');
while ($themeRow = $smcFunc['db_fetch_assoc']($themeResult)) {
	$replacement = smfConfigureRewriteURL($themeRow['value'], $previousForumURL, $forumURL, $themeRow['variable']);
	if ($replacement !== null && $replacement !== $themeRow['value']) {
		$themeChanges[] = array(
			'id_theme' => (int) $themeRow['id_theme'],
			'variable' => $themeRow['variable'],
			'previous' => $themeRow['value'],
			'replacement' => $replacement,
		);
	}
}
$smcFunc['db_free_result']($themeResult);

$boardURLChanged = $previousForumURL !== $forumURL;
if (!$boardURLChanged && empty($settingChanges) && empty($themeChanges)) {
	fwrite(STDOUT, 'unchanged');
	exit(0);
}

$settingsFileChanged = false;
$transactionOpen = false;
$configurationCommitted = false;
$changedThemeIDs = array();
try {
	if ($boardURLChanged) {
		if (!updateSettingsFile(array('boardurl' => $forumURL))) {
			throw new RuntimeException('Could not update the persisted Settings.php board URL.');
		}
		$settingsFileChanged = true;
	}

	if (!empty($settingChanges) || !empty($themeChanges)) {
		smfConfigureQueryOrFail('START TRANSACTION', array(), 'Could not begin the SMF URL configuration transaction.');
		$transactionOpen = true;

		foreach ($settingChanges as $change) {
			smfConfigureQueryOrFail('
				UPDATE {db_prefix}settings
				SET value = {string:replacement}
				WHERE variable = {string:variable}
					AND value = {string:previous}', array(
				'replacement' => $change['replacement'],
				'variable' => $change['variable'],
				'previous' => $change['previous'],
			), 'Could not update the SMF setting ' . $change['variable'] . '.');
			if ($smcFunc['db_affected_rows']() !== 1) {
				throw new RuntimeException('The SMF setting ' . $change['variable'] . ' changed concurrently.');
			}
		}

		foreach ($themeChanges as $change) {
			smfConfigureQueryOrFail('
				UPDATE {db_prefix}themes
				SET value = {string:replacement}
				WHERE id_theme = {int:id_theme}
					AND id_member = {int:global_member}
					AND variable = {string:variable}
					AND value = {string:previous}', array(
				'replacement' => $change['replacement'],
				'id_theme' => $change['id_theme'],
				'global_member' => 0,
				'variable' => $change['variable'],
				'previous' => $change['previous'],
			), 'Could not update an SMF global theme URL setting.');
			if ($smcFunc['db_affected_rows']() !== 1) {
				throw new RuntimeException('An SMF global theme URL setting changed concurrently.');
			}
			$changedThemeIDs[$change['id_theme']] = true;
		}

		if (!empty($themeChanges)) {
			if (!array_key_exists('settings_updated', $modSettings)) {
				throw new RuntimeException('SMF is missing the settings_updated configuration value.');
			}
			$settingsUpdated = max(time(), (int) $modSettings['settings_updated'] + 1);
			smfConfigureQueryOrFail('
				UPDATE {db_prefix}settings
				SET value = {string:replacement}
				WHERE variable = {string:variable}
					AND value = {string:previous}', array(
				'replacement' => (string) $settingsUpdated,
				'variable' => 'settings_updated',
				'previous' => $modSettings['settings_updated'],
			), 'Could not invalidate SMF theme settings.');
			if ($smcFunc['db_affected_rows']() !== 1) {
				throw new RuntimeException('SMF theme settings changed concurrently.');
			}
		}

		foreach ($settingChanges as $change) {
			smfConfigureVerifySetting($change);
		}
		foreach ($themeChanges as $change) {
			smfConfigureVerifyThemeSetting($change);
		}

		smfConfigureQueryOrFail('COMMIT', array(), 'Could not commit the SMF URL configuration transaction.');
		$transactionOpen = false;
	}
	$configurationCommitted = true;
} catch (Throwable $error) {
	if ($transactionOpen) {
		smfConfigureQueryOrFail('ROLLBACK', array(), 'Could not roll back the SMF URL configuration transaction.');
	}
	if (!$configurationCommitted && $settingsFileChanged && !updateSettingsFile(array('boardurl' => $previousForumURL))) {
		smfConfigureURLFail('Could not restore Settings.php after a failed SMF URL configuration update: ' . $error->getMessage());
	}
	smfConfigureURLFail('Could not synchronize SMF URL configuration: ' . $error->getMessage());
}

if (!empty($settingChanges) || !empty($themeChanges)) {
	cache_put_data('modSettings', null, 90);
}
foreach (array_keys($changedThemeIDs) as $themeID) {
	cache_put_data('theme_settings-' . $themeID, null, 90);
}

fwrite(STDOUT, 'changed boardurl=' . ($boardURLChanged ? '1' : '0') . ' settings=' . count($settingChanges) . ' themes=' . count($themeChanges));
