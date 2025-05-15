
import "module2/my_module2" for Helper

Helper.call()

class Guy {
    foreign CallToGo(x, y, z, w)
    static Test() { System.print("static test from guy, let's take it away")}
    SomethingElse(say, sayit) { 
        // sayit = 1.3215 * 4
        System.print("sayit: %(say)") 
    }
    construct new() { System.print("Argh! I mean, hi as a Guy!")}
}

var standalone = Fn.new {|anarg, anarg2| 
    System.print("a standalone function?")
    System.print(anarg)
    System.print(anarg2)
}

var guy = Guy.new()

System.print(guy.CallToGo({"one":1, "two":2}, 0, 0, 0))

var testFunc = Fn.new { |test| System.print("This is a %(test) function.") }