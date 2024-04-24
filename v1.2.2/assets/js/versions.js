async function getVersions(lang) {
    response = await fetch('/firefly/v1.2.2/assets/js/versions.json');
    versions = await response.json();

    versionDropdown = document.getElementById('versionDropdown');

    versions.forEach(version => {
        versionLink = document.createElement('a');
        url = '/firefly/' + version;
        versionLink.href = url;
        versionLink.innerText = version;
        versionDropdown.appendChild(versionLink)
    });
}