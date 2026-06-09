// Renders the blocked domain from the query string.
// The value is untrusted, so it is only ever assigned via textContent.
(function () {
    const params = new URLSearchParams(window.location.search);
    const domain = params.get('domain');

    if (domain) {
        document.getElementById('domain').textContent = domain;
    }

    document.getElementById('back').addEventListener('click', () => {
        // Prefer going back; fall back to closing the tab's content.
        if (window.history.length > 1) {
            window.history.back();
        } else {
            window.location.href = 'about:blank';
        }
    });
})();
