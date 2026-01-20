(function() {
  const CONSENT_KEY = 'lwcn_cookie_consent';
  const CONSENT_VERSION = '1'; // Increment to reset consent for all users
  const consentBanner = document.getElementById('cookie-consent');
  const consentOverlay = document.getElementById('cookie-consent-overlay');
  const acceptBtn = document.getElementById('cookie-accept');
  const declineBtn = document.getElementById('cookie-decline');

  // Use both localStorage and cookies for maximum compatibility
  function getConsent() {
    // Try localStorage first
    try {
      const stored = localStorage.getItem(CONSENT_KEY);
      if (stored) {
        const data = JSON.parse(stored);
        if (data.version === CONSENT_VERSION) {
          return data.consent;
        }
      }
    } catch (e) {}

    // Fallback to cookie
    const match = document.cookie.match(new RegExp('(^| )' + CONSENT_KEY + '=([^;]+)'));
    if (match) {
      try {
        const data = JSON.parse(decodeURIComponent(match[2]));
        if (data.version === CONSENT_VERSION) {
          return data.consent;
        }
      } catch (e) {}
    }

    return null;
  }

  function setConsent(value) {
    const data = {
      consent: value,
      version: CONSENT_VERSION,
      timestamp: new Date().toISOString()
    };

    // Store in localStorage
    try {
      localStorage.setItem(CONSENT_KEY, JSON.stringify(data));
    } catch (e) {}

    // Also store as cookie (expires in 1 year)
    const expires = new Date();
    expires.setFullYear(expires.getFullYear() + 1);
    document.cookie = CONSENT_KEY + '=' + encodeURIComponent(JSON.stringify(data)) +
      ';expires=' + expires.toUTCString() +
      ';path=/;SameSite=Lax;Secure';
  }

  function loadGoogleAnalytics() {
    const gaId = window.LWCN_GA_ID;
    if (!gaId || gaId === '' || gaId.startsWith('G-XXXX')) {
      return;
    }

    // Prevent double loading
    if (window.gaLoaded) return;
    window.gaLoaded = true;

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
    if (consentOverlay) {
      consentOverlay.style.display = 'block';
      document.body.style.overflow = 'hidden'; // Prevent scrolling
    }
  }

  function hideBanner() {
    if (consentBanner) {
      consentBanner.style.display = 'none';
    }
    if (consentOverlay) {
      consentOverlay.style.display = 'none';
      document.body.style.overflow = ''; // Re-enable scrolling
    }
  }

  // Check existing consent on page load
  const consent = getConsent();

  if (consent === 'accepted') {
    loadGoogleAnalytics();
    hideBanner();
  } else if (consent === 'declined') {
    hideBanner();
  } else {
    // No consent yet - show banner
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

  // Global function to revoke cookie consent (can be called from privacy page)
  window.LWCN_revokeConsent = function() {
    // Clear localStorage
    try {
      localStorage.removeItem(CONSENT_KEY);
    } catch (e) {}

    // Clear cookie
    document.cookie = CONSENT_KEY + '=;expires=Thu, 01 Jan 1970 00:00:00 GMT;path=/;SameSite=Lax;Secure';

    // Show banner again
    showBanner();

    // Disable GA if it was loaded (user needs to reload for full effect)
    if (window.gtag) {
      window['ga-disable-' + window.LWCN_GA_ID] = true;
    }

    return true;
  };

  // Get current consent status
  window.LWCN_getConsentStatus = function() {
    return getConsent();
  };
})();
