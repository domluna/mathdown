MathDown
========

Watches markdown files in a directory. Converts markdown to html to be displayed
in browser on edit.

### Rendering math

To render math prefix the math expression with `rendermath`, ex:

```sh
rendermath f(x) = \int_{-\infty}^\infty
    \hat f(\xi)\,e^{2 \pi i \xi x}
    \,d\xi
```

### TODO

* Better styling, option for BYOS? (Bring your own styles).
* Add a view option, for static viewing.
