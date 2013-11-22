---
layout: default
---
# I love Golang!

{% for p in site.posts %}
<a href="{{ p.url }}">{{ p.title }}</a><br />
{% endfor %}
