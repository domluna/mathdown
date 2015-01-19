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
* Reading option, essentially what this will do find all .md files
  and generate a nice viewing template in the browser. Even better if
  we could fetch md files from something like a github repo.
