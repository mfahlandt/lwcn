(function() {
  const CONSENT_KEY = 'lwcn_cookie_consent';
  const consentBanner = document.getElementById('cookie-consent');
  const acceptBtn = document.getElementById('cookie-accept');
  const declineBtn = document.getElementById('cookie-decline');

  function getConsent() {
    return localStorage.getItem(CONSENT_KEY);
  }

  function setConsent(value) {
    localStorage.setItem(CONSENT_KEY, value);
  }

  function loadGoogleAnalytics() {
    const gaId = window.LWCN_GA_ID;
    if (!gaId || gaId === '' || gaId.startsWith('G-XXXX')) {
      return;
    }

    // Load gtag.js
    const script = document.createElement('script');
    script.async = true;
    script.src = 'https://www.googletagmanager.com/gtag/js?id=' + gaId;
    document.head.appendChild(script);

    // Initialize gtag
    window.dataLayer = window.dataLayer || [];
    function gtag(){dataLayer.push(arguments);}
    gtag('js', new Date());
    gtag('config', gaId, { 'anonymize_ip': true });
    window.gtag = gtag;
  }

  function showBanner() {
    if (consentBanner) {
      consentBanner.style.display = 'block';
    }
  }

  function hideBanner() {
    if (consentBanner) {
      consentBanner.style.display = 'none';
    }
  }

  // Check existing consent
  const consent = getConsent();

  if (consent === 'accepted') {
    loadGoogleAnalytics();
  } else if (consent === null) {
    showBanner();
  }

  // Event listeners
  if (acceptBtn) {
    acceptBtn.addEventListener('click', function() {
      setConsent('accepted');
      hideBanner();
      loadGoogleAnalytics();
    });
  }

  if (declineBtn) {
    declineBtn.addEventListener('click', function() {
      setConsent('declined');
      hideBanner();
    });
  }
})();
