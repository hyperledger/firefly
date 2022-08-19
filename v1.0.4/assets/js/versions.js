async function getVersions(lang) {
    response = await fetch('/firefly/assets/js/versions.json');
    versions = await response.json();

    versionDropdown = document.getElementById('versionDropdown');

    versions.forEach(version => {
        versionLink = document.createElement('a');
        url = '/firefly/';
        if (version != 'latest') {
            url += version + '/' ;
        }
        if (lang != null && lang != '' && lang != 'en') {
            url += lang + '/';
        }
        versionLink.href = url;
        versionLink.innerText = version;
        versionDropdown.appendChild(versionLink)
    });
}