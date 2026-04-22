package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"whatsapp-sales-os-enterprise/backend/internal/models"
)

type LandingService struct {
	AI *AIService
}

func NewLandingService(ai *AIService) *LandingService {
	return &LandingService{AI: ai}
}

func (s *LandingService) GenerateLanding(
	ctx context.Context,
	bot models.Bot,
	cfg models.BotConfig,
	in models.LandingPage,
) (models.LandingPage, error) {
	if bot.Phone == "" {
		return models.LandingPage{}, fmt.Errorf("el bot no tiene número conectado")
	}

	whatsappURL := fmt.Sprintf(
		"https://wa.me/%s?text=Hola%%20quiero%%20informaci%C3%B3n",
		bot.Phone,
	)

	stylePreset := strings.TrimSpace(in.StylePreset)
	if stylePreset == "" {
		stylePreset = "dark_premium"
	}

	primaryColor := strings.TrimSpace(in.PrimaryColor)
	if primaryColor == "" {
		primaryColor = "#2563eb"
	}

	secondaryColor := strings.TrimSpace(in.SecondaryColor)
	if secondaryColor == "" {
		secondaryColor = "#0f172a"
	}

	logoURL := strings.TrimSpace(in.LogoURL)
	faviconURL := strings.TrimSpace(in.FaviconURL)
	heroImageURL := strings.TrimSpace(in.HeroImageURL)
	youtubeURL := toYouTubeEmbedURL(in.YoutubeURL)
	facebookPixelID := strings.TrimSpace(in.FacebookPixelID)
	googleAnalytics := strings.TrimSpace(in.GoogleAnalytics)

	trackingMode := strings.TrimSpace(strings.ToLower(in.TrackingMode))
	if trackingMode == "" {
		trackingMode = "auto"
	}
	trackingBaseURL := strings.TrimSpace(in.TrackingBaseURL)
	trackingURL := resolveTrackingURL(trackingMode, trackingBaseURL)

	videoInstruction := "NO incluir video."
	if in.ShowVideo && youtubeURL != "" {
		videoInstruction = "INCLUIR obligatoriamente una sección principal de video, protagonista, bien visible y ubicada cerca del hero usando exactamente esta URL embebible: " + youtubeURL + ". No usar títulos genéricos como 'Explora nuestro video'; el título debe estar contextualizado al negocio."
	}

	imageInstruction := "NO incluir imagen principal."
	if in.ShowImage && heroImageURL != "" {
		imageInstruction = "INCLUIR obligatoriamente una imagen principal destacada, estética y bien visible usando exactamente esta URL: " + heroImageURL
	}

	logoInstruction := "NO incluir logo."
	if logoURL != "" {
		logoInstruction = "INCLUIR obligatoriamente el logo en navbar y/o hero usando exactamente esta URL: " + logoURL
	}

	faviconInstruction := "NO incluir favicon."
	if faviconURL != "" {
		faviconInstruction = "INCLUIR obligatoriamente un favicon dentro de <head> usando exactamente esta URL: " + faviconURL
	}

	pixelInstruction := "NO incluir Facebook Pixel."
	if facebookPixelID != "" {
		pixelInstruction = "INCLUIR el script de Facebook Pixel en <head> usando este ID: " + facebookPixelID
	}

	analyticsInstruction := "NO incluir Google Analytics."
	if googleAnalytics != "" {
		analyticsInstruction = "INCLUIR el script de Google Analytics en <head> usando este ID: " + googleAnalytics
	}

	system := fmt.Sprintf(`
Eres un experto senior en CRO, UX/UI, copywriting de conversión y desarrollo frontend premium.

Tu tarea es generar una landing page MUY PROFESIONAL en HTML5 válido, con estética moderna, orientada a conversión y totalmente responsive.

REQUISITOS OBLIGATORIOS:
- devolver SOLO HTML, sin markdown, sin explicación
- incluir <!DOCTYPE html>
- incluir <html>, <head>, <body>
- usar TailwindCSS CDN
- responsive real y perfecto en móvil, tablet y desktop
- diseño premium, limpio y moderno
- navbar profesional
- hero section muy impactante
- propuesta de valor fuerte
- beneficios bien presentados
- testimonios creíbles en formato slider/carrusel visual atractivo
- sección CTA fuerte
- sección FAQ con mínimo 5 preguntas y respuestas
- la sección FAQ debe mostrarse como acordeón visual moderno
- footer profesional
- el footer debe mostrar automáticamente el año actual
- contrastes correctos y legibles
- NO dejar textos invisibles ni fondos que dañen la lectura
- el HTML debe renderizar bien al abrirse como archivo local .html
- evitar dependencias extrañas fuera de TailwindCDN
- el botón flotante de WhatsApp no lo dibujes tú; ya será inyectado después por el sistema
- debes dejar espacio visual suficiente para que el botón flotante no tape CTAs importantes
- si hay video, debe verse protagonista, no escondido ni cortado
- si hay video, colócalo alto en la landing, como sección principal o semiprincipal
- no uses títulos genéricos para el video; contextualízalo al negocio
- las secciones deben verse profesionales también en móviles
- usar microanimaciones suaves si es posible con CSS simple
- mantener una jerarquía visual clara y premium

INSTRUCCIONES CRÍTICAS:
- %s
- %s
- %s
- %s
- %s
- %s

ESTILO VISUAL:
- STYLE_PRESET: %s
- PRIMARY_COLOR: %s
- SECONDARY_COLOR: %s

NEGOCIO:
%s

DESCRIPCIÓN:
%s

OFERTA:
%s

PÚBLICO:
%s

PROMPT DEL USUARIO:
%s

WHATSAPP_URL:
%s

LOGO_URL:
%s

FAVICON_URL:
%s

HERO_IMAGE_URL:
%s

YOUTUBE_URL:
%s

FACEBOOK_PIXEL_ID:
%s

GOOGLE_ANALYTICS:
%s
`,
		videoInstruction,
		imageInstruction,
		logoInstruction,
		faviconInstruction,
		pixelInstruction,
		analyticsInstruction,
		stylePreset,
		primaryColor,
		secondaryColor,
		cfg.BusinessName,
		cfg.BusinessDescription,
		cfg.Offer,
		cfg.TargetAudience,
		in.Prompt,
		whatsappURL,
		logoURL,
		faviconURL,
		heroImageURL,
		youtubeURL,
		facebookPixelID,
		googleAnalytics,
	)

	resp, err := s.AI.GenerateHTML(ctx, system, cfg.Model)
	if err != nil {
		return models.LandingPage{}, err
	}

	html := cleanHTML(resp)
	landingID := uuid.NewString()

	html = enforceDynamicYear(html)
	html = injectLandingViewTracking(html, trackingURL, bot.ClientID, bot.ID, landingID)
	html = injectWhatsAppButtonTracked(html, trackingURL, whatsappURL, bot.ClientID, bot.ID, landingID)
	html = injectFavicon(html, faviconURL)

	now := time.Now()

	return models.LandingPage{
		ID:              landingID,
		ClientID:        bot.ClientID,
		BotID:           bot.ID,
		Name:            defaultLandingName(in.Name, cfg.BusinessName),
		Prompt:          in.Prompt,
		Status:          "generated",
		StylePreset:     stylePreset,
		LogoURL:         logoURL,
		FaviconURL:      faviconURL,
		HeroImageURL:    heroImageURL,
		YoutubeURL:      youtubeURL,
		FacebookPixelID: facebookPixelID,
		GoogleAnalytics: googleAnalytics,
		PrimaryColor:    primaryColor,
		SecondaryColor:  secondaryColor,
		ShowVideo:       in.ShowVideo,
		ShowImage:       in.ShowImage,
		Html:            html,
		PreviewHTML:     html,
		WhatsappURL:     whatsappURL,
		TrackingMode:    trackingMode,
		TrackingBaseURL: trackingBaseURL,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func defaultLandingName(name, business string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	business = strings.TrimSpace(business)
	if business == "" {
		return "Landing generada"
	}
	return "Landing - " + business
}

func cleanHTML(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "```html", "")
	s = strings.ReplaceAll(s, "```HTML", "")
	s = strings.ReplaceAll(s, "```", "")
	return strings.TrimSpace(s)
}

func toYouTubeEmbedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.Contains(raw, "youtube.com/embed/") {
		return raw
	}

	if strings.Contains(raw, "youtu.be/") {
		parts := strings.Split(raw, "youtu.be/")
		if len(parts) > 1 {
			id := parts[1]
			if i := strings.Index(id, "?"); i >= 0 {
				id = id[:i]
			}
			return "https://www.youtube.com/embed/" + id
		}
	}

	if strings.Contains(raw, "watch?v=") {
		parts := strings.Split(raw, "watch?v=")
		if len(parts) > 1 {
			id := parts[1]
			if i := strings.Index(id, "&"); i >= 0 {
				id = id[:i]
			}
			return "https://www.youtube.com/embed/" + id
		}
	}

	return raw
}

func resolveTrackingURL(mode, base string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	base = strings.TrimSpace(base)

	if mode == "" {
		mode = "auto"
	}

	if mode == "external" && base != "" {
		base = strings.TrimRight(base, "/")
		return base + "/api/public/funnel/event"
	}

	return "/api/public/funnel/event"
}

func injectFavicon(html, faviconURL string) string {
	faviconURL = strings.TrimSpace(faviconURL)
	if faviconURL == "" {
		return html
	}

	tag := fmt.Sprintf(`<link rel="icon" type="image/png" href="%s">`, faviconURL)
	lower := strings.ToLower(html)

	if strings.Contains(lower, `rel="icon"`) || strings.Contains(lower, "rel='icon'") {
		return html
	}

	if strings.Contains(lower, "</head>") {
		return strings.Replace(html, "</head>", tag+"\n</head>", 1)
	}

	return html
}

func enforceDynamicYear(html string) string {
	lower := strings.ToLower(html)

	script := `
<script>
(function(){
  var el = document.getElementById('current-year');
  var year = new Date().getFullYear();
  if (el) el.textContent = year;
})();
</script>
`

	if strings.Contains(lower, `id="current-year"`) {
		if strings.Contains(lower, "</body>") {
			return strings.Replace(html, "</body>", script+"\n</body>", 1)
		}
		return html + script
	}

	yearFooter := `
<script>
(function(){
  var footers = document.querySelectorAll('footer');
  var year = new Date().getFullYear();
  footers.forEach(function(f){
    if (!f.innerHTML.match(/20\d{2}/)) {
      f.innerHTML = f.innerHTML + '<div style="margin-top:12px;font-size:14px;opacity:.85;">&copy; <span id="current-year">'+year+'</span> Todos los derechos reservados.</div>';
    }
  });
})();
</script>
`

	if strings.Contains(lower, "</body>") {
		return strings.Replace(html, "</body>", yearFooter+"\n</body>", 1)
	}
	return html + yearFooter
}

func injectLandingViewTracking(html, trackingURL, clientID, botID, landingID string) string {
	script := fmt.Sprintf(`
<script>
(function() {
  try {
    fetch(%q, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        client_id: %q,
        bot_id: %q,
        landing_id: %q,
        event_type: 'landing_view',
        metadata: window.location.href
      })
    });
  } catch(e) {}
})();
</script>
`, trackingURL, clientID, botID, landingID)

	if strings.Contains(strings.ToLower(html), "</body>") {
		return strings.Replace(html, "</body>", script+"</body>", 1)
	}
	return html + script
}

func injectWhatsAppButtonTracked(html, trackingURL, whatsappURL, clientID, botID, landingID string) string {
	if strings.TrimSpace(whatsappURL) == "" {
		return html
	}

	button := fmt.Sprintf(`
<a href="%s"
   target="_blank"
   rel="noopener noreferrer"
   aria-label="WhatsApp"
   onclick='try {
     fetch(%q, {
       method: "POST",
       headers: {"Content-Type":"application/json"},
       body: JSON.stringify({
         client_id: %q,
         bot_id: %q,
         landing_id: %q,
         event_type: "whatsapp_click",
         metadata: window.location.href
       })
     });
   } catch(e) {}'
   style="
      position:fixed;
      right:20px;
      bottom:20px;
      width:68px;
      height:68px;
      border-radius:9999px;
      background:#25D366;
      color:#fff;
      display:flex;
      align-items:center;
      justify-content:center;
      text-decoration:none;
      font-size:32px;
      font-weight:bold;
      box-shadow:0 14px 35px rgba(0,0,0,.28);
      z-index:99999;
      border:4px solid #ffffff;
   ">
   <span style="font-family:Arial,sans-serif;line-height:1;">✆</span>
</a>
<style>
@media (max-width: 768px) {
  a[aria-label="WhatsApp"] {
    width: 60px !important;
    height: 60px !important;
    right: 16px !important;
    bottom: 16px !important;
    font-size: 28px !important;
  }
}
</style>
`, whatsappURL, trackingURL, clientID, botID, landingID)

	if strings.Contains(strings.ToLower(html), "</body>") {
		return strings.Replace(html, "</body>", button+"</body>", 1)
	}
	return html + button
}