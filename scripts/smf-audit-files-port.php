<?php

// Read-only exhaustive file audit for a running SMF container. It scans every
// readable regular file reachable below /var/www/html for the exact TCP port
// supplied by SMF_AUDIT_PORT. It does not write, delete, or follow directory
// symlinks. Matches in binary files are included.

function smfFileAuditFail($message)
{
	fwrite(STDERR, $message . "\n");
	exit(1);
}

function smfFileAuditPattern($port)
{
	return '/(?<![0-9]):' . preg_quote((string) $port, '/') . '(?![0-9])/';
}

function smfFileAuditExcerpt($buffer, $matchOffset, $matchLength)
{
	$startOffset = max(0, $matchOffset - 240);
	$endOffset = min(strlen($buffer), $matchOffset + $matchLength + 1024);
	$excerpt = substr($buffer, $startOffset, $endOffset - $startOffset);
	if ($startOffset > 0) {
		$excerpt = '...' . $excerpt;
	}
	if ($endOffset < strlen($buffer)) {
		$excerpt .= '...';
	}

	return $excerpt;
}

function smfFileAuditRedactPHPAssignments($excerpt)
{
	return preg_replace_callback(
		'#(\$(?:webmaster_email|auth_secret)\s*=\s*)([\'\"]).*?\2#s',
		function ($match) {
			return $match[1] . $match[2] . '****' . $match[2];
		},
		$excerpt
	);
}

function smfFileAuditPrintMatch($path, $byteOffset, $excerpt)
{
	global $smfFileAuditMatchCount;

	$smfFileAuditMatchCount++;
	fwrite(
		STDOUT,
		$path . "\t" . $byteOffset . "\t" .
		json_encode(smfFileAuditRedactPHPAssignments($excerpt), JSON_UNESCAPED_SLASHES | JSON_INVALID_UTF8_SUBSTITUTE) . "\n"
	);
}

function smfFileAuditScanFile($path, $port)
{
	global $smfFileAuditByteCount;

	$handle = @fopen($path, 'rb');
	if ($handle === false) {
		fwrite(STDERR, 'Skipping unreadable file: ' . $path . "\n");
		return;
	}

	$chunkSize = 65536;
	$carrySize = 2048;
	$carry = '';
	$fileOffset = 0;
	$pattern = smfFileAuditPattern($port);
	while (!feof($handle)) {
		$chunk = fread($handle, $chunkSize);
		if ($chunk === false) {
			fclose($handle);
			smfFileAuditFail('Could not read file: ' . $path);
		}
		if ($chunk === '' && !feof($handle)) {
			fclose($handle);
			smfFileAuditFail('Could not make progress while reading file: ' . $path);
		}

		$buffer = $carry . $chunk;
		$bufferOffset = $fileOffset - strlen($carry);
		$isLastBuffer = feof($handle);
		$processEnd = $isLastBuffer ? strlen($buffer) : max(0, strlen($buffer) - $carrySize);
		if ($processEnd > 0 && preg_match_all($pattern, $buffer, $matches, PREG_OFFSET_CAPTURE)) {
			foreach ($matches[0] as $match) {
				$matchOffset = $match[1];
				if ($matchOffset >= $processEnd) {
					continue;
				}
				smfFileAuditPrintMatch(
					$path,
					$bufferOffset + $matchOffset,
					smfFileAuditExcerpt($buffer, $matchOffset, strlen($match[0]))
				);
			}
		}

		$smfFileAuditByteCount += strlen($chunk);
		$fileOffset += strlen($chunk);
		$carry = strlen($buffer) > $carrySize ? substr($buffer, -$carrySize) : $buffer;
	}
	fclose($handle);
}

$port = getenv('SMF_AUDIT_PORT');
if (!is_string($port) || !preg_match('/^[0-9]+$/', $port) || (int) $port < 1 || (int) $port > 65535) {
	smfFileAuditFail('SMF_AUDIT_PORT must be a TCP port from 1 through 65535.');
}
$port = (int) $port;
$rootPath = '/var/www/html';
if (!is_dir($rootPath) || !is_readable($rootPath)) {
	smfFileAuditFail('The SMF document root is not readable: ' . $rootPath);
}

$smfFileAuditFileCount = 0;
$smfFileAuditByteCount = 0;
$smfFileAuditMatchCount = 0;
fwrite(STDOUT, "path\tbyte_offset\texcerpt\n");
try {
	$iterator = new RecursiveIteratorIterator(
		new RecursiveDirectoryIterator($rootPath, FilesystemIterator::SKIP_DOTS),
		RecursiveIteratorIterator::LEAVES_ONLY
	);
	foreach ($iterator as $fileInfo) {
		if (!$fileInfo->isFile() || !$fileInfo->isReadable()) {
			continue;
		}
		$smfFileAuditFileCount++;
		smfFileAuditScanFile($fileInfo->getPathname(), $port);
	}
} catch (UnexpectedValueException $error) {
	smfFileAuditFail('Could not enumerate the SMF document root: ' . $error->getMessage());
}

fwrite(
	STDOUT,
	"Scanned " . $smfFileAuditFileCount . " files and " . $smfFileAuditByteCount .
	" bytes; found " . $smfFileAuditMatchCount . " occurrences of TCP port " . $port . ".\n"
);
