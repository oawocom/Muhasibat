# Bu qovluq artıq məcburi deyil.
#
# Caddy `tls internal` ilə origin sertifikatını AVTOMATİK yaradır
# (bax: caddy/Caddyfile). Əl ilə sertifikat qoymağa ehtiyac yoxdur.
# Cloudflare SSL/TLS rejimini "Full" seçmək kifayətdir.
#
# (İstəsən, əl ilə Cloudflare Origin Certificate də istifadə edə bilərsən:
#  origin.pem + origin.key fayllarını bura qoy və Caddyfile-da `tls internal`
#  sətrini `tls /etc/caddy/certs/origin.pem /etc/caddy/certs/origin.key` ilə
#  əvəz et, Cloudflare rejimini "Full (strict)" seç.)
