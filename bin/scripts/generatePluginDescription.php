<?php
<<<CONFIG
packages:
    - "arhframe/yamlarh: 1.*"
CONFIG;
// you need melody: https://github.com/sensiolabs/melody

$template = __DIR__ . '/../template/plugin-description.yml';
use Arhframe\Yamlarh\Yamlarh;
use Symfony\Component\Yaml\Yaml;

$output = __DIR__ . '/../../repo-index.yml';

if (is_file($output)) {
    $currentData = Yaml::parse(file_get_contents($output));
    $date = date('Y-m-d', $currentData['plugins']['created']);
} else {
    $date = date('Y-m-d');
}
$date_updated = date('Y-m-d');

$yamlarh = new Yamlarh($template);
$data = $yamlarh->parse();


$binaries = &$data['plugins']['binaries'];

foreach ($binaries as &$binary) {
    $path = pathinfo($binary['url'], PATHINFO_BASENAME);
    $checksum = sha1_file(__DIR__ . '/../../out/' . $path);
    $binary['checksum'] = $checksum;
}
$dataString = Yaml::dump($data, 10, 2);

//sanitize just for repo-index.yml in cloudfoundry community plugin repo
$dataString = str_replace("'", "", $dataString);
$dataString = str_replace("null", "", $dataString);
$dataString = str_replace("\n    -", "", $dataString);
$dataString = str_replace("  platform:", "- platform:", $dataString);
$dataString = str_replace("authors:\n      name:", "authors:\n    - name:", $dataString);
/*******************************/
file_put_contents($output, $dataString);

echo "Description generated in 'repo-index.yml'";