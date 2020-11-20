package configx

func n() {
	/*
		// Use the POSIX compliant pflag lib instead of Go's flag lib.
		f := flag.NewFlagSet("config", flag.ContinueOnError)
		f.Usage = func() {
			fmt.Println(f.FlagUsages())
			os.Exit(0)
		}
		// Path to one or more config files to load into koanf along with some config params.
		f.StringSlice("conf", []string{"mock/mock.toml"}, "path to one or more .toml config files")
		f.String("time", "2020-01-01", "a time string")
		f.String("type", "xxx", "type of the app")
		f.Parse(os.Args[1:])

		// Load the config files provided in the commandline.
		cFiles, _ := f.GetStringSlice("conf")
		for _, c := range cFiles {
			if err := k.Load(file.Provider(c), toml.Parser()); err != nil {
				log.Fatalf("error loading file: %v", err)
			}
		}

		// "time" and "type" may have been loaded from the config file, but
		// they can still be overridden with the values from the command line.
		// The bundled posflag.Provider takes a flagset from the spf13/pflag lib.
		// Passing the Koanf instance to posflag helps it deal with default command
		// line flag values that are not present in conf maps from previously loaded
		// providers.
		if err := k.Load(posflag.Provider(f, ".", k), nil); err != nil {
			log.Fatalf("error loading config: %v", err)
		}
	*/
}
